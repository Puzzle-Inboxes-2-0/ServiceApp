package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lib/pq"
)

// ReservedIP represents a reserved IP in the database
type ReservedIP struct {
	ID                  int       `json:"id"`
	IPAddress           string    `json:"ip_address"`
	ReservationBlockID  string    `json:"reservation_block_id"`
	UID                 string    `json:"uid"`
	Location            string    `json:"location"`
	Status              string    `json:"status"`
	IsBlacklisted       bool      `json:"is_blacklisted"`
	BlacklistDetails    []string  `json:"blacklist_details"`
	ReservedAt          time.Time `json:"reserved_at"`
	LastCheckedAt       *time.Time `json:"last_checked_at,omitempty"`
	ReleasedAt          *time.Time `json:"released_at,omitempty"`
	AssignedTo          *string   `json:"assigned_to,omitempty"`
	UsageCount          int       `json:"usage_count"`
	Metadata            map[string]interface{} `json:"metadata,omitempty"`
	Notes               *string   `json:"notes,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// ReservationAttempt represents an IP reservation attempt
type ReservationAttempt struct {
	ID              int       `json:"id"`
	AttemptUID      string    `json:"attempt_uid"`
	BlockID         *string   `json:"block_id,omitempty"`
	IPAddress       *string   `json:"ip_address,omitempty"`
	Location        string    `json:"location"`
	Success         bool      `json:"success"`
	FailureReason   *string   `json:"failure_reason,omitempty"`
	WasBlacklisted  bool      `json:"was_blacklisted"`
	BlacklistsFound []string  `json:"blacklists_found"`
	AttemptedAt     time.Time `json:"attempted_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	DurationMs      *int      `json:"duration_ms,omitempty"`
	ActionTaken     *string   `json:"action_taken,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// BlacklistHistoryEntry represents a blacklist check in history
type BlacklistHistoryEntry struct {
	ID              int       `json:"id"`
	ReservedIPID    int       `json:"reserved_ip_id"`
	IPAddress       string    `json:"ip_address"`
	CheckedAt       time.Time `json:"checked_at"`
	WasBlacklisted  bool      `json:"was_blacklisted"`
	BlacklistsFound []string  `json:"blacklists_found"`
	CheckDurationMs int       `json:"check_duration_ms"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// QuotaSnapshot represents a point-in-time quota snapshot
type QuotaSnapshot struct {
	ID               int       `json:"id"`
	TotalBlocks      int       `json:"total_blocks"`
	EstimatedLimit   int       `json:"estimated_limit"`
	Remaining        int       `json:"remaining"`
	ProtectedBlocks  int       `json:"protected_blocks"`
	SingleIPBlocks   int       `json:"single_ip_blocks"`
	Location         *string   `json:"location,omitempty"`
	SnapshotAt       time.Time `json:"snapshot_at"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// CreateReservedIP inserts a new reserved IP into the database
func CreateReservedIP(ip *ReservedIP) error {
	blacklistJSON, err := json.Marshal(ip.BlacklistDetails)
	if err != nil {
		return fmt.Errorf("failed to marshal blacklist details: %w", err)
	}

	metadataJSON, err := json.Marshal(ip.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO reserved_ips 
		(ip_address, reservation_block_id, uid, location, status, is_blacklisted, 
		 blacklist_details, reserved_at, assigned_to, usage_count, metadata, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at, updated_at
	`

	err = DB.QueryRow(
		query,
		ip.IPAddress,
		ip.ReservationBlockID,
		ip.UID,
		ip.Location,
		ip.Status,
		ip.IsBlacklisted,
		blacklistJSON,
		ip.ReservedAt,
		ip.AssignedTo,
		ip.UsageCount,
		metadataJSON,
		ip.Notes,
	).Scan(&ip.ID, &ip.CreatedAt, &ip.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create reserved IP: %w", err)
	}

	return nil
}

// GetReservedIPByID retrieves a reserved IP by ID
func GetReservedIPByID(id int) (*ReservedIP, error) {
	query := `
		SELECT id, ip_address, reservation_block_id, uid, location, status, 
		       is_blacklisted, blacklist_details, reserved_at, last_checked_at, 
		       released_at, assigned_to, usage_count, metadata, notes, 
		       created_at, updated_at
		FROM reserved_ips
		WHERE id = $1
	`

	var ip ReservedIP
	var blacklistJSON []byte
	var metadataJSON []byte

	err := DB.QueryRow(query, id).Scan(
		&ip.ID,
		&ip.IPAddress,
		&ip.ReservationBlockID,
		&ip.UID,
		&ip.Location,
		&ip.Status,
		&ip.IsBlacklisted,
		&blacklistJSON,
		&ip.ReservedAt,
		&ip.LastCheckedAt,
		&ip.ReleasedAt,
		&ip.AssignedTo,
		&ip.UsageCount,
		&metadataJSON,
		&ip.Notes,
		&ip.CreatedAt,
		&ip.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("reserved IP not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get reserved IP: %w", err)
	}

	if err := json.Unmarshal(blacklistJSON, &ip.BlacklistDetails); err != nil {
		return nil, fmt.Errorf("failed to unmarshal blacklist details: %w", err)
	}

	if err := json.Unmarshal(metadataJSON, &ip.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &ip, nil
}

// GetReservedIPByAddress retrieves a reserved IP by IP address
func GetReservedIPByAddress(ipAddress string) (*ReservedIP, error) {
	query := `
		SELECT id, ip_address, reservation_block_id, uid, location, status, 
		       is_blacklisted, blacklist_details, reserved_at, last_checked_at, 
		       released_at, assigned_to, usage_count, metadata, notes, 
		       created_at, updated_at
		FROM reserved_ips
		WHERE ip_address = $1
	`

	var ip ReservedIP
	var blacklistJSON []byte
	var metadataJSON []byte

	err := DB.QueryRow(query, ipAddress).Scan(
		&ip.ID,
		&ip.IPAddress,
		&ip.ReservationBlockID,
		&ip.UID,
		&ip.Location,
		&ip.Status,
		&ip.IsBlacklisted,
		&blacklistJSON,
		&ip.ReservedAt,
		&ip.LastCheckedAt,
		&ip.ReleasedAt,
		&ip.AssignedTo,
		&ip.UsageCount,
		&metadataJSON,
		&ip.Notes,
		&ip.CreatedAt,
		&ip.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("reserved IP not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get reserved IP: %w", err)
	}

	if err := json.Unmarshal(blacklistJSON, &ip.BlacklistDetails); err != nil {
		return nil, fmt.Errorf("failed to unmarshal blacklist details: %w", err)
	}

	if err := json.Unmarshal(metadataJSON, &ip.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &ip, nil
}

// ListReservedIPs retrieves all reserved IPs with optional filtering
func ListReservedIPs(status *string, isBlacklisted *bool, location *string) ([]ReservedIP, error) {
	query := `
		SELECT id, ip_address, reservation_block_id, uid, location, status, 
		       is_blacklisted, blacklist_details, reserved_at, last_checked_at, 
		       released_at, assigned_to, usage_count, metadata, notes, 
		       created_at, updated_at
		FROM reserved_ips
		WHERE 1=1
	`
	args := []interface{}{}
	argCount := 1

	if status != nil {
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, *status)
		argCount++
	}

	if isBlacklisted != nil {
		query += fmt.Sprintf(" AND is_blacklisted = $%d", argCount)
		args = append(args, *isBlacklisted)
		argCount++
	}

	if location != nil {
		query += fmt.Sprintf(" AND location = $%d", argCount)
		args = append(args, *location)
		argCount++
	}

	query += " ORDER BY reserved_at DESC"

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query reserved IPs: %w", err)
	}
	defer rows.Close()

	var ips []ReservedIP
	for rows.Next() {
		var ip ReservedIP
		var blacklistJSON []byte
		var metadataJSON []byte

		err := rows.Scan(
			&ip.ID,
			&ip.IPAddress,
			&ip.ReservationBlockID,
			&ip.UID,
			&ip.Location,
			&ip.Status,
			&ip.IsBlacklisted,
			&blacklistJSON,
			&ip.ReservedAt,
			&ip.LastCheckedAt,
			&ip.ReleasedAt,
			&ip.AssignedTo,
			&ip.UsageCount,
			&metadataJSON,
			&ip.Notes,
			&ip.CreatedAt,
			&ip.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan reserved IP: %w", err)
		}

		if err := json.Unmarshal(blacklistJSON, &ip.BlacklistDetails); err != nil {
			return nil, fmt.Errorf("failed to unmarshal blacklist details: %w", err)
		}

		if err := json.Unmarshal(metadataJSON, &ip.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		ips = append(ips, ip)
	}

	return ips, nil
}

// UpdateReservedIPStatus updates the status of a reserved IP
func UpdateReservedIPStatus(id int, status string, assignedTo *string) error {
	query := `
		UPDATE reserved_ips
		SET status = $1, assigned_to = $2, updated_at = NOW()
		WHERE id = $3
	`

	result, err := DB.Exec(query, status, assignedTo, id)
	if err != nil {
		return fmt.Errorf("failed to update reserved IP status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("reserved IP not found")
	}

	return nil
}

// UpdateReservedIPBlacklistStatus updates the blacklist status
func UpdateReservedIPBlacklistStatus(id int, isBlacklisted bool, blacklists []string) error {
	blacklistJSON, err := json.Marshal(blacklists)
	if err != nil {
		return fmt.Errorf("failed to marshal blacklists: %w", err)
	}

	query := `
		UPDATE reserved_ips
		SET is_blacklisted = $1, blacklist_details = $2, 
		    last_checked_at = NOW(), updated_at = NOW()
		WHERE id = $3
	`

	result, err := DB.Exec(query, isBlacklisted, blacklistJSON, id)
	if err != nil {
		return fmt.Errorf("failed to update blacklist status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("reserved IP not found")
	}

	return nil
}

// DeleteReservedIP deletes a reserved IP
func DeleteReservedIP(id int) error {
	query := `DELETE FROM reserved_ips WHERE id = $1`

	result, err := DB.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete reserved IP: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("reserved IP not found")
	}

	return nil
}

// CreateReservationAttempt records an IP reservation attempt
func CreateReservationAttempt(attempt *ReservationAttempt) error {
	blacklistJSON, err := json.Marshal(attempt.BlacklistsFound)
	if err != nil {
		return fmt.Errorf("failed to marshal blacklists: %w", err)
	}

	metadataJSON, err := json.Marshal(attempt.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO ip_reservation_attempts 
		(attempt_uid, block_id, ip_address, location, success, failure_reason,
		 was_blacklisted, blacklists_found, attempted_at, completed_at, 
		 duration_ms, action_taken, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id
	`

	err = DB.QueryRow(
		query,
		attempt.AttemptUID,
		attempt.BlockID,
		attempt.IPAddress,
		attempt.Location,
		attempt.Success,
		attempt.FailureReason,
		attempt.WasBlacklisted,
		blacklistJSON,
		attempt.AttemptedAt,
		attempt.CompletedAt,
		attempt.DurationMs,
		attempt.ActionTaken,
		metadataJSON,
	).Scan(&attempt.ID)

	if err != nil {
		// Check for unique constraint violation
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return fmt.Errorf("attempt UID already exists: %w", err)
		}
		return fmt.Errorf("failed to create reservation attempt: %w", err)
	}

	return nil
}

// CreateBlacklistHistoryEntry records a blacklist check
func CreateBlacklistHistoryEntry(entry *BlacklistHistoryEntry) error {
	blacklistJSON, err := json.Marshal(entry.BlacklistsFound)
	if err != nil {
		return fmt.Errorf("failed to marshal blacklists: %w", err)
	}

	metadataJSON, err := json.Marshal(entry.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO reserved_ip_blacklist_history 
		(reserved_ip_id, ip_address, checked_at, was_blacklisted, 
		 blacklists_found, check_duration_ms, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	err = DB.QueryRow(
		query,
		entry.ReservedIPID,
		entry.IPAddress,
		entry.CheckedAt,
		entry.WasBlacklisted,
		blacklistJSON,
		entry.CheckDurationMs,
		metadataJSON,
	).Scan(&entry.ID)

	if err != nil {
		return fmt.Errorf("failed to create blacklist history entry: %w", err)
	}

	return nil
}

// CreateQuotaSnapshot records a quota snapshot
func CreateQuotaSnapshot(snapshot *QuotaSnapshot) error {
	metadataJSON, err := json.Marshal(snapshot.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO ionos_quota_snapshots 
		(total_blocks, estimated_limit, remaining, protected_blocks, 
		 single_ip_blocks, location, snapshot_at, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	err = DB.QueryRow(
		query,
		snapshot.TotalBlocks,
		snapshot.EstimatedLimit,
		snapshot.Remaining,
		snapshot.ProtectedBlocks,
		snapshot.SingleIPBlocks,
		snapshot.Location,
		snapshot.SnapshotAt,
		metadataJSON,
	).Scan(&snapshot.ID)

	if err != nil {
		return fmt.Errorf("failed to create quota snapshot: %w", err)
	}

	return nil
}

// GetReservationStatistics returns statistics about IP reservations
func GetReservationStatistics() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Count by status
	statusQuery := `
		SELECT status, COUNT(*) as count
		FROM reserved_ips
		GROUP BY status
	`
	rows, err := DB.Query(statusQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query status counts: %w", err)
	}
	defer rows.Close()

	statusCounts := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan status count: %w", err)
		}
		statusCounts[status] = count
	}
	stats["by_status"] = statusCounts

	// Count blacklisted
	var blacklistedCount int
	err = DB.QueryRow("SELECT COUNT(*) FROM reserved_ips WHERE is_blacklisted = true").Scan(&blacklistedCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count blacklisted IPs: %w", err)
	}
	stats["blacklisted_count"] = blacklistedCount

	// Total count
	var totalCount int
	err = DB.QueryRow("SELECT COUNT(*) FROM reserved_ips").Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count total IPs: %w", err)
	}
	stats["total_count"] = totalCount

	return stats, nil
}

