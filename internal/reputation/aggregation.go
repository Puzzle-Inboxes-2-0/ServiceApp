package reputation

import (
	"fmt"
	"sync"
	"time"

	"golang-backend-service/internal/database"
	"golang-backend-service/internal/logger"

	"github.com/sirupsen/logrus"
)

// AggregationService handles periodic IP reputation aggregation
type AggregationService struct {
	config        ReputationConfig
	ticker        *time.Ticker
	stopChan      chan bool
	running       bool
	mu            sync.Mutex
	lastRun       time.Time
	ipsProcessed  int
	errors        int
}

// NewAggregationService creates a new aggregation service
func NewAggregationService(config ReputationConfig) *AggregationService {
	return &AggregationService{
		config:   config,
		stopChan: make(chan bool),
		running:  false,
	}
}

// Start begins the aggregation service
func (s *AggregationService) Start(intervalMinutes int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("aggregation service is already running")
	}

	s.ticker = time.NewTicker(time.Duration(intervalMinutes) * time.Minute)
	s.running = true

	logger.WithFields(logrus.Fields{
		"action":           "aggregation_service_start",
		"interval_minutes": intervalMinutes,
		"window_minutes":   s.config.WindowMinutes,
	}).Info("Starting IP reputation aggregation service")

	// Run immediately on start
	go s.runAggregation()

	// Start periodic aggregation
	go func() {
		for {
			select {
			case <-s.ticker.C:
				s.runAggregation()
			case <-s.stopChan:
				logger.Info("Aggregation service stopped")
				return
			}
		}
	}()

	return nil
}

// Stop stops the aggregation service
func (s *AggregationService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	if s.ticker != nil {
		s.ticker.Stop()
	}

	s.stopChan <- true
	s.running = false

	logger.WithFields(logrus.Fields{
		"action": "aggregation_service_stop",
	}).Info("Stopping IP reputation aggregation service")
}

// IsRunning returns whether the service is running
func (s *AggregationService) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// GetStats returns service statistics
func (s *AggregationService) GetStats() map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	return map[string]interface{}{
		"running":       s.running,
		"last_run":      s.lastRun,
		"ips_processed": s.ipsProcessed,
		"errors":        s.errors,
	}
}

// runAggregation performs the aggregation process
func (s *AggregationService) runAggregation() {
	start := time.Now()

	logger.WithFields(logrus.Fields{
		"action": "aggregation_run_start",
	}).Info("Starting IP reputation aggregation run")

	// Get IPs that need aggregation
	since := time.Now().Add(-time.Duration(s.config.WindowMinutes) * time.Minute)
	ips, err := database.GetIPsNeedingAggregation(since)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"action": "aggregation_get_ips_failed",
			"error":  err.Error(),
		}).Error("Failed to get IPs needing aggregation")
		s.mu.Lock()
		s.errors++
		s.mu.Unlock()
		return
	}

	logger.WithFields(logrus.Fields{
		"action":    "aggregation_ips_found",
		"ip_count":  len(ips),
	}).Info("Found IPs needing aggregation")

	// Process each IP
	successCount := 0
	errorCount := 0

	for _, ip := range ips {
		if err := s.aggregateIPMetrics(ip); err != nil {
			logger.WithFields(logrus.Fields{
				"action": "aggregation_ip_failed",
				"ip":     ip,
				"error":  err.Error(),
			}).Error("Failed to aggregate metrics for IP")
			errorCount++
		} else {
			successCount++
		}
	}

	// Update stats
	s.mu.Lock()
	s.lastRun = time.Now()
	s.ipsProcessed += successCount
	s.errors += errorCount
	s.mu.Unlock()

	// Record metrics
	RecordAggregationRun("success", successCount)
	if errorCount > 0 {
		RecordAggregationRun("failed", errorCount)
	}

	duration := time.Since(start)

	logger.WithFields(logrus.Fields{
		"action":       "aggregation_run_complete",
		"ips_success":  successCount,
		"ips_failed":   errorCount,
		"duration_ms":  duration.Milliseconds(),
	}).Info("IP reputation aggregation run completed")
}

// aggregateIPMetrics aggregates metrics for a single IP
func (s *AggregationService) aggregateIPMetrics(ip string) error {
	windowStart := time.Now().Add(-time.Duration(s.config.WindowMinutes) * time.Minute)
	windowEnd := time.Now()

	// Get previous metrics to determine old status
	oldMetrics, err := database.GetIPReputationMetrics(ip)
	var oldStatus string
	if err == nil {
		oldStatus = oldMetrics.Status
	} else {
		oldStatus = "unknown"
	}

	// For this implementation, we'll use a simple estimate for total_sent
	// In production, this should come from your sending system
	totalSent := s.estimateTotalSent(ip, windowStart)

	// Calculate health metrics
	health, err := CalculateIPHealthCheck(ip, s.config.WindowMinutes, totalSent)
	if err != nil {
		return fmt.Errorf("failed to calculate health check: %w", err)
	}

	// Determine status
	status := DetermineIPStatus(*health, s.config)

	// Record rejection ratio metric
	RecordRejectionRatio(health.RejectionRatio)

	// Create metrics record
	metrics := &database.IPReputationMetrics{
		IP:                       ip,
		WindowStart:              windowStart,
		WindowEnd:                windowEnd,
		TotalSent:                health.TotalSent,
		TotalRejected:            health.TotalRejected,
		RejectionRatio:           health.RejectionRatio,
		UniqueDomainsRejected:    health.UniqueDomainsRejected,
		DistinctRejectionReasons: health.ReputationCodes,
		MajorProvidersRejecting:  health.MajorProviders,
		Status:                   status,
		LastUpdated:              time.Now(),
		Metadata: map[string]interface{}{
			"throttle_count": health.ThrottleCount,
			"domain_counts":  health.DomainCounts,
			"issue_type":     GetIssueType(*health),
		},
	}

	// Save metrics
	if err := database.UpsertIPReputationMetrics(metrics); err != nil {
		return fmt.Errorf("failed to save metrics: %w", err)
	}

	// If status changed, record action and take appropriate measures
	if oldStatus != status && oldStatus != "unknown" {
		// Record status change metric
		RecordStatusChange(ip, oldStatus, status)
		
		if err := s.handleStatusChange(ip, oldStatus, status, *health); err != nil {
			logger.WithFields(logrus.Fields{
				"action": "status_change_handler_failed",
				"ip":     ip,
				"error":  err.Error(),
			}).Error("Failed to handle status change")
		}
	} else {
		// Update gauge even if status didn't change
		RecordStatusChange(ip, status, status)
	}

	logger.WithFields(logrus.Fields{
		"action":          "ip_metrics_aggregated",
		"ip":              ip,
		"status":          status,
		"old_status":      oldStatus,
		"rejection_ratio": health.RejectionRatio,
		"total_rejected":  health.TotalRejected,
	}).Info("IP metrics aggregated successfully")

	return nil
}

// handleStatusChange handles actions when IP status changes
func (s *AggregationService) handleStatusChange(ip, oldStatus, newStatus string, health IPHealthCheck) error {
	// Record the action
	action := &database.IPAction{
		IP:             ip,
		Action:         "status_change",
		PreviousStatus: oldStatus,
		NewStatus:      newStatus,
		Reason:         GetStatusSummary(newStatus, health),
		TriggeredBy:    "automated_aggregation",
		Metadata: map[string]interface{}{
			"rejection_ratio":       health.RejectionRatio,
			"unique_domains":        health.UniqueDomainsRejected,
			"major_providers":       health.MajorProviders,
			"total_rejected":        health.TotalRejected,
		},
		CreatedAt: time.Now(),
	}

	if err := database.InsertIPAction(action); err != nil {
		return fmt.Errorf("failed to record action: %w", err)
	}

	// Take automated actions based on new status
	switch newStatus {
	case "blacklisted":
		s.handleBlacklistedIP(ip, health)
	case "quarantine":
		s.handleQuarantinedIP(ip, health)
	case "warning":
		s.handleWarningIP(ip, health)
	}

	return nil
}

// handleBlacklistedIP handles critical blacklist status
func (s *AggregationService) handleBlacklistedIP(ip string, health IPHealthCheck) {
	logger.WithFields(logrus.Fields{
		"action":          "ip_blacklisted",
		"ip":              ip,
		"rejection_ratio": health.RejectionRatio,
		"major_providers": health.MajorProviders,
	}).Error("IP has been BLACKLISTED - immediate action required")

	// Trigger DNSBL check asynchronously
	CheckDNSBLAsync(ip, 5, func(result *DNSBLResult, err error) {
		if err != nil {
			logger.WithFields(logrus.Fields{
				"action": "dnsbl_check_failed",
				"ip":     ip,
				"error":  err.Error(),
			}).Error("DNSBL check failed")
			return
		}

		if result.Listed {
			logger.WithFields(logrus.Fields{
				"action":   "dnsbl_listings_found",
				"ip":       ip,
				"listings": result.Listings,
			}).Error("IP is listed on DNSBLs")
		}
	})

	// In production, you would:
	// - Send critical alerts to ops team
	// - Automatically quarantine the IP
	// - Swap to backup IP
	// - Trigger incident response
}

// handleQuarantinedIP handles quarantine status
func (s *AggregationService) handleQuarantinedIP(ip string, health IPHealthCheck) {
	logger.WithFields(logrus.Fields{
		"action":          "ip_quarantined",
		"ip":              ip,
		"rejection_ratio": health.RejectionRatio,
	}).Warn("IP has been QUARANTINED - investigation needed")

	// Trigger DNSBL check
	CheckDNSBLAsync(ip, 5, func(result *DNSBLResult, err error) {
		if err == nil && result.Listed {
			logger.WithFields(logrus.Fields{
				"action":   "dnsbl_listings_found_quarantine",
				"ip":       ip,
				"listings": result.Listings,
			}).Warn("Quarantined IP is also listed on DNSBLs")
		}
	})

	// In production, you would:
	// - Send warning alerts
	// - Reduce traffic by 50%
	// - Increase monitoring
}

// handleWarningIP handles warning status
func (s *AggregationService) handleWarningIP(ip string, health IPHealthCheck) {
	logger.WithFields(logrus.Fields{
		"action":          "ip_warning",
		"ip":              ip,
		"rejection_ratio": health.RejectionRatio,
	}).Warn("IP has WARNING status - monitor closely")

	// In production, you would:
	// - Send informational alerts
	// - Increase monitoring frequency
	// - Prepare for potential escalation
}

// estimateTotalSent estimates total emails sent (placeholder implementation)
// In production, this should query your actual sending metrics
func (s *AggregationService) estimateTotalSent(ip string, since time.Time) int {
	// This is a simple estimation based on failures
	// In production, integrate with your actual sending system metrics
	
	failures, err := database.GetSMTPFailuresByIP(ip, since)
	if err != nil {
		return 100 // Default minimum for assessment
	}

	failureCount := len(failures)
	
	// For testing/demo: Use a more conservative ratio
	// Assume 5% failure rate for estimation
	// So if we have X failures, total sent is approximately X / 0.05
	estimated := int(float64(failureCount) * 20) // failureCount / 0.05 = failureCount * 20
	
	// Ensure minimum volume for assessment
	if estimated < s.config.MinVolumeForAssessment {
		return s.config.MinVolumeForAssessment
	}

	return estimated
}

// AggregateIPOnDemand manually triggers aggregation for a specific IP
func AggregateIPOnDemand(ip string, config ReputationConfig) (*database.IPReputationMetrics, error) {
	service := NewAggregationService(config)
	
	if err := service.aggregateIPMetrics(ip); err != nil {
		return nil, fmt.Errorf("failed to aggregate metrics: %w", err)
	}

	// Retrieve and return the updated metrics
	metrics, err := database.GetIPReputationMetrics(ip)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve metrics: %w", err)
	}

	return metrics, nil
}

