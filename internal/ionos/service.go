package ionos

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang-backend-service/internal/database"
)

// Service handles IP reservation business logic
type Service struct {
	client         *Client
	blacklistCheck *DNSBLChecker
	logger         *logrus.Logger
	defaultLocation string
	maxQuota       int
}

// NewService creates a new IP reservation service
func NewService(client *Client, logger *logrus.Logger, defaultLocation string, maxQuota int) *Service {
	return &Service{
		client:          client,
		blacklistCheck:  NewDNSBLChecker(logger),
		logger:          logger,
		defaultLocation: defaultLocation,
		maxQuota:        maxQuota,
	}
}

// ReserveIPRequest represents a request to reserve IPs
type ReserveIPRequest struct {
	Count    int    `json:"count"`
	Location string `json:"location"`
}

// ReserveIPResponse represents the response from reserving IPs
type ReserveIPResponse struct {
	SuccessCount    int                       `json:"success_count"`
	FailureCount    int                       `json:"failure_count"`
	BlacklistedCount int                      `json:"blacklisted_count"`
	ReservedIPs     []database.ReservedIP     `json:"reserved_ips"`
	Attempts        []database.ReservationAttempt `json:"attempts,omitempty"`
}

// ReserveCleanIPs reserves a specified number of clean (non-blacklisted) IPs
func (s *Service) ReserveCleanIPs(ctx context.Context, count int, location string) (*ReserveIPResponse, error) {
	if location == "" {
		location = s.defaultLocation
	}

	s.logger.WithFields(logrus.Fields{
		"action":   "reserve_clean_ips",
		"count":    count,
		"location": location,
	}).Info("Starting IP reservation process")

	// Check quota before starting
	quota, err := s.CheckQuota(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check quota: %w", err)
	}

	if quota.Remaining < count {
		return nil, fmt.Errorf("insufficient quota: need %d, have %d", count, quota.Remaining)
	}

	response := &ReserveIPResponse{
		ReservedIPs: []database.ReservedIP{},
		Attempts:    []database.ReservationAttempt{},
	}

	successCount := 0
	maxAttempts := count * 5 // Allow up to 5x attempts to account for blacklisted IPs

	for attempt := 0; attempt < maxAttempts && successCount < count; attempt++ {
		select {
		case <-ctx.Done():
			return response, fmt.Errorf("context cancelled: %w", ctx.Err())
		default:
		}

		s.logger.WithFields(logrus.Fields{
			"attempt":  attempt + 1,
			"success":  successCount,
			"target":   count,
		}).Info("Attempting IP reservation")

		// Reserve single IP
		result, err := s.reserveSingleIP(ctx, location)
		if err != nil {
			s.logger.WithError(err).Error("Failed to reserve IP")
			response.FailureCount++
			continue
		}

		// Record attempt
		response.Attempts = append(response.Attempts, *result.Attempt)

		if result.IsClean {
			response.ReservedIPs = append(response.ReservedIPs, *result.ReservedIP)
			successCount++
			response.SuccessCount++
		} else {
			response.BlacklistedCount++
		}

		// Rate limiting
		time.Sleep(1 * time.Second)
	}

	s.logger.WithFields(logrus.Fields{
		"action":            "reserve_clean_ips",
		"success_count":     response.SuccessCount,
		"blacklisted_count": response.BlacklistedCount,
		"failure_count":     response.FailureCount,
	}).Info("IP reservation process completed")

	return response, nil
}

// reserveIPResult contains the result of a single IP reservation
type reserveIPResult struct {
	IsClean    bool
	ReservedIP *database.ReservedIP
	Attempt    *database.ReservationAttempt
}

// reserveSingleIP reserves a single IP, checks blacklist, and handles accordingly
func (s *Service) reserveSingleIP(ctx context.Context, location string) (*reserveIPResult, error) {
	startTime := time.Now()
	attemptUID := uuid.New().String()[:8]
	blockName := fmt.Sprintf("IP-Reserver-%s", attemptUID)

	attempt := &database.ReservationAttempt{
		AttemptUID:  attemptUID,
		Location:    location,
		AttemptedAt: startTime,
		Metadata:    make(map[string]interface{}),
	}

	// Reserve IP block from IONOS
	block, err := s.client.ReserveIPBlock(ctx, location, 1, blockName)
	if err != nil {
		attempt.Success = false
		failureReason := err.Error()
		attempt.FailureReason = &failureReason
		completedAt := time.Now()
		attempt.CompletedAt = &completedAt
		duration := int(time.Since(startTime).Milliseconds())
		attempt.DurationMs = &duration

		// Record failed attempt
		if dbErr := database.CreateReservationAttempt(attempt); dbErr != nil {
			s.logger.WithError(dbErr).Error("Failed to record reservation attempt")
		}

		return nil, fmt.Errorf("failed to reserve IP block: %w", err)
	}

	blockID := block.ID
	attempt.BlockID = &blockID

	// Wait for IPs to be assigned if not immediately available
	if len(block.Properties.IPs) == 0 {
		s.logger.Info("IPs not immediately available, waiting...")
		time.Sleep(5 * time.Second)
		block, err = s.client.GetIPBlock(ctx, blockID)
		if err != nil {
			attempt.Success = false
			failureReason := fmt.Sprintf("failed to retrieve block: %s", err.Error())
			attempt.FailureReason = &failureReason
			completedAt := time.Now()
			attempt.CompletedAt = &completedAt
			duration := int(time.Since(startTime).Milliseconds())
			attempt.DurationMs = &duration

			// Record failed attempt
			if dbErr := database.CreateReservationAttempt(attempt); dbErr != nil {
				s.logger.WithError(dbErr).Error("Failed to record reservation attempt")
			}

			return nil, fmt.Errorf("failed to retrieve IP block: %w", err)
		}
	}

	if len(block.Properties.IPs) == 0 {
		attempt.Success = false
		failureReason := "no IPs assigned to block"
		attempt.FailureReason = &failureReason
		completedAt := time.Now()
		attempt.CompletedAt = &completedAt
		duration := int(time.Since(startTime).Milliseconds())
		attempt.DurationMs = &duration

		// Record failed attempt
		if dbErr := database.CreateReservationAttempt(attempt); dbErr != nil {
			s.logger.WithError(dbErr).Error("Failed to record reservation attempt")
		}

		return nil, fmt.Errorf("no IPs assigned to block %s", blockID)
	}

	ipAddress := block.Properties.IPs[0]
	attempt.IPAddress = &ipAddress

	s.logger.WithFields(logrus.Fields{
		"ip":       ipAddress,
		"block_id": blockID,
	}).Info("IP block reserved, checking blacklist")

	// Check blacklist
	blacklistResult, err := s.blacklistCheck.CheckIP(ctx, ipAddress)
	if err != nil {
		s.logger.WithError(err).Error("Failed to check blacklist")
		// Don't fail the reservation if blacklist check fails, just mark as unknown
		blacklistResult = &BlacklistResult{
			IP:            ipAddress,
			IsBlacklisted: false,
			Blacklists:    []string{},
		}
	}

	attempt.WasBlacklisted = blacklistResult.IsBlacklisted
	attempt.BlacklistsFound = blacklistResult.Blacklists
	completedAt := time.Now()
	attempt.CompletedAt = &completedAt
	duration := int(time.Since(startTime).Milliseconds())
	attempt.DurationMs = &duration

	if blacklistResult.IsBlacklisted {
		s.logger.WithFields(logrus.Fields{
			"ip":         ipAddress,
			"blacklists": blacklistResult.Blacklists,
		}).Warn("IP is blacklisted, deleting")

		// Delete the dirty IP
		if err := s.client.DeleteIPBlock(ctx, blockID); err != nil {
			s.logger.WithError(err).Error("Failed to delete blacklisted IP block")
		}

		attempt.Success = false
		actionTaken := "deleted"
		attempt.ActionTaken = &actionTaken

		// Record attempt
		if err := database.CreateReservationAttempt(attempt); err != nil {
			s.logger.WithError(err).Error("Failed to record reservation attempt")
		}

		return &reserveIPResult{
			IsClean:    false,
			ReservedIP: nil,
			Attempt:    attempt,
		}, nil
	}

	// IP is clean, store in database
	s.logger.WithField("ip", ipAddress).Info("IP is clean, storing in database")

	reservedIP := &database.ReservedIP{
		IPAddress:          ipAddress,
		ReservationBlockID: blockID,
		UID:                attemptUID,
		Location:           location,
		Status:             "reserved",
		IsBlacklisted:      false,
		BlacklistDetails:   []string{},
		ReservedAt:         startTime,
		UsageCount:         0,
		Metadata:           make(map[string]interface{}),
	}

	if err := database.CreateReservedIP(reservedIP); err != nil {
		s.logger.WithError(err).Error("Failed to store reserved IP in database")
		
		attempt.Success = false
		failureReason := fmt.Sprintf("database error: %s", err.Error())
		attempt.FailureReason = &failureReason
		actionTaken := "kept_but_not_stored"
		attempt.ActionTaken = &actionTaken

		// Record attempt
		if dbErr := database.CreateReservationAttempt(attempt); dbErr != nil {
			s.logger.WithError(dbErr).Error("Failed to record reservation attempt")
		}

		return nil, fmt.Errorf("failed to store reserved IP: %w", err)
	}

	// Record blacklist check in history
	historyEntry := &database.BlacklistHistoryEntry{
		ReservedIPID:    reservedIP.ID,
		IPAddress:       ipAddress,
		CheckedAt:       time.Now(),
		WasBlacklisted:  false,
		BlacklistsFound: []string{},
		CheckDurationMs: int(blacklistResult.CheckDuration.Milliseconds()),
		Metadata:        make(map[string]interface{}),
	}

	if err := database.CreateBlacklistHistoryEntry(historyEntry); err != nil {
		s.logger.WithError(err).Error("Failed to record blacklist history")
	}

	attempt.Success = true
	actionTaken := "kept"
	attempt.ActionTaken = &actionTaken

	// Record successful attempt
	if err := database.CreateReservationAttempt(attempt); err != nil {
		s.logger.WithError(err).Error("Failed to record reservation attempt")
	}

	return &reserveIPResult{
		IsClean:    true,
		ReservedIP: reservedIP,
		Attempt:    attempt,
	}, nil
}

// QuotaInfo contains information about IONOS quota
type QuotaInfo struct {
	TotalBlocks      int `json:"total_blocks"`
	ProtectedBlocks  int `json:"protected_blocks"`
	SingleIPBlocks   int `json:"single_ip_blocks"`
	EstimatedLimit   int `json:"estimated_limit"`
	Remaining        int `json:"remaining"`
}

// CheckQuota checks the current IONOS quota usage
func (s *Service) CheckQuota(ctx context.Context) (*QuotaInfo, error) {
	blocks, err := s.client.ListIPBlocks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list IP blocks: %w", err)
	}

	quota := &QuotaInfo{
		TotalBlocks:    len(blocks.Items),
		EstimatedLimit: s.maxQuota,
	}

	// Identify protected blocks (11-IP blocks)
	for _, block := range blocks.Items {
		size := block.Properties.Size
		ipCount := len(block.Properties.IPs)

		if size == 11 || ipCount == 11 {
			quota.ProtectedBlocks++
		} else if size == 1 || ipCount == 1 {
			quota.SingleIPBlocks++
		}
	}

	quota.Remaining = quota.EstimatedLimit - quota.TotalBlocks

	s.logger.WithFields(logrus.Fields{
		"total_blocks":     quota.TotalBlocks,
		"protected_blocks": quota.ProtectedBlocks,
		"single_ip_blocks": quota.SingleIPBlocks,
		"remaining":        quota.Remaining,
	}).Info("Quota check completed")

	// Record quota snapshot
	snapshot := &database.QuotaSnapshot{
		TotalBlocks:     quota.TotalBlocks,
		EstimatedLimit:  quota.EstimatedLimit,
		Remaining:       quota.Remaining,
		ProtectedBlocks: quota.ProtectedBlocks,
		SingleIPBlocks:  quota.SingleIPBlocks,
		SnapshotAt:      time.Now(),
		Metadata:        make(map[string]interface{}),
	}

	if err := database.CreateQuotaSnapshot(snapshot); err != nil {
		s.logger.WithError(err).Error("Failed to record quota snapshot")
	}

	return quota, nil
}

// CleanupSingleIPBlocks removes all single-IP blocks (except those in use)
func (s *Service) CleanupSingleIPBlocks(ctx context.Context) (int, error) {
	s.logger.Info("Starting cleanup of single-IP blocks")

	blocks, err := s.client.ListIPBlocks(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to list IP blocks: %w", err)
	}

	// Get all reserved IPs from database to avoid deleting in-use blocks
	reservedIPs, err := database.ListReservedIPs(nil, nil, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to list reserved IPs: %w", err)
	}

	inUseBlocks := make(map[string]bool)
	for _, ip := range reservedIPs {
		if ip.Status == "in_use" || ip.Status == "reserved" {
			inUseBlocks[ip.ReservationBlockID] = true
		}
	}

	deletedCount := 0
	for _, block := range blocks.Items {
		// Skip protected blocks (11-IP blocks)
		if block.Properties.Size == 11 || len(block.Properties.IPs) == 11 {
			continue
		}

		// Skip in-use blocks
		if inUseBlocks[block.ID] {
			continue
		}

		// Delete single-IP blocks
		if block.Properties.Size == 1 || len(block.Properties.IPs) == 1 {
			s.logger.WithFields(logrus.Fields{
				"block_id": block.ID,
				"name":     block.Properties.Name,
			}).Info("Deleting single-IP block")

			if err := s.client.DeleteIPBlock(ctx, block.ID); err != nil {
				s.logger.WithError(err).Error("Failed to delete block")
				continue
			}

			deletedCount++
			time.Sleep(300 * time.Millisecond) // Rate limiting
		}
	}

	s.logger.WithField("deleted_count", deletedCount).Info("Cleanup completed")
	return deletedCount, nil
}

// RecheckBlacklist rechecks an IP against blacklists
func (s *Service) RecheckBlacklist(ctx context.Context, ipID int) error {
	ip, err := database.GetReservedIPByID(ipID)
	if err != nil {
		return fmt.Errorf("failed to get reserved IP: %w", err)
	}

	s.logger.WithField("ip", ip.IPAddress).Info("Rechecking blacklist status")

	result, err := s.blacklistCheck.CheckIP(ctx, ip.IPAddress)
	if err != nil {
		return fmt.Errorf("failed to check blacklist: %w", err)
	}

	// Update database
	if err := database.UpdateReservedIPBlacklistStatus(ipID, result.IsBlacklisted, result.Blacklists); err != nil {
		return fmt.Errorf("failed to update blacklist status: %w", err)
	}

	// Record in history
	historyEntry := &database.BlacklistHistoryEntry{
		ReservedIPID:    ipID,
		IPAddress:       ip.IPAddress,
		CheckedAt:       time.Now(),
		WasBlacklisted:  result.IsBlacklisted,
		BlacklistsFound: result.Blacklists,
		CheckDurationMs: int(result.CheckDuration.Milliseconds()),
		Metadata:        make(map[string]interface{}),
	}

	if err := database.CreateBlacklistHistoryEntry(historyEntry); err != nil {
		s.logger.WithError(err).Error("Failed to record blacklist history")
	}

	s.logger.WithFields(logrus.Fields{
		"ip":             ip.IPAddress,
		"is_blacklisted": result.IsBlacklisted,
		"blacklists":     result.Blacklists,
	}).Info("Blacklist recheck completed")

	return nil
}

