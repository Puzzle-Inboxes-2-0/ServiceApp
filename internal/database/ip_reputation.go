package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// SMTPFailure represents an individual SMTP delivery failure
type SMTPFailure struct {
	ID              int       `json:"id"`
	SendingIP       string    `json:"sending_ip"`
	RecipientEmail  string    `json:"recipient_email"`
	RecipientDomain string    `json:"recipient_domain"`
	SMTPCode        int       `json:"smtp_code"`
	EnhancedCode    string    `json:"enhanced_code"`
	Reason          string    `json:"reason"`
	MXServer        string    `json:"mx_server"`
	Timestamp       time.Time `json:"timestamp"`
	EventID         string    `json:"event_id"`
	AttemptNumber   int       `json:"attempt_number"`
}

// IPReputationMetrics represents aggregated reputation metrics for an IP
type IPReputationMetrics struct {
	ID                       int                    `json:"id"`
	IP                       string                 `json:"ip"`
	WindowStart              time.Time              `json:"window_start"`
	WindowEnd                time.Time              `json:"window_end"`
	TotalSent                int                    `json:"total_sent"`
	TotalRejected            int                    `json:"total_rejected"`
	RejectionRatio           float64                `json:"rejection_ratio"`
	UniqueDomainsRejected    int                    `json:"unique_domains_rejected"`
	DistinctRejectionReasons map[string]int         `json:"distinct_rejection_reasons"`
	MajorProvidersRejecting  []string               `json:"major_providers_rejecting"`
	Status                   string                 `json:"status"`
	LastUpdated              time.Time              `json:"last_updated"`
	Metadata                 map[string]interface{} `json:"metadata"`
}

// DNSBLCheck represents a DNSBL check result
type DNSBLCheck struct {
	ID              int       `json:"id"`
	IP              string    `json:"ip"`
	CheckedAt       time.Time `json:"checked_at"`
	Listed          bool      `json:"listed"`
	Listings        []string  `json:"listings"`
	CheckDurationMS int       `json:"check_duration_ms"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// IPAction represents an action taken on an IP
type IPAction struct {
	ID             int                    `json:"id"`
	IP             string                 `json:"ip"`
	Action         string                 `json:"action"`
	PreviousStatus string                 `json:"previous_status"`
	NewStatus      string                 `json:"new_status"`
	Reason         string                 `json:"reason"`
	TriggeredBy    string                 `json:"triggered_by"`
	Metadata       map[string]interface{} `json:"metadata"`
	CreatedAt      time.Time              `json:"created_at"`
}

// InsertSMTPFailure inserts a new SMTP failure record
// Uses ON CONFLICT DO NOTHING to handle duplicate webhook events gracefully
func InsertSMTPFailure(failure *SMTPFailure) error {
	query := `
		INSERT INTO smtp_failures (
			sending_ip, recipient_email, recipient_domain, smtp_code, 
			enhanced_code, reason, mx_server, timestamp, event_id, attempt_number
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (event_id) DO NOTHING
		RETURNING id
	`

	err := DB.QueryRow(
		query,
		failure.SendingIP,
		failure.RecipientEmail,
		failure.RecipientDomain,
		failure.SMTPCode,
		failure.EnhancedCode,
		failure.Reason,
		failure.MXServer,
		failure.Timestamp,
		failure.EventID,
		failure.AttemptNumber,
	).Scan(&failure.ID)

	// If ON CONFLICT triggered, no rows returned - this is OK (duplicate event)
	if err != nil {
		// Check if it's a "no rows" error (duplicate), which is expected
		if err.Error() == "sql: no rows in result set" {
			return nil // Duplicate event, silently ignore
		}
		return fmt.Errorf("failed to insert SMTP failure: %w", err)
	}

	return nil
}

// GetSMTPFailuresByIP retrieves SMTP failures for a specific IP within a time window
func GetSMTPFailuresByIP(ip string, since time.Time) ([]SMTPFailure, error) {
	query := `
		SELECT id, sending_ip, recipient_email, recipient_domain, smtp_code,
		       enhanced_code, reason, mx_server, timestamp, event_id, attempt_number
		FROM smtp_failures
		WHERE sending_ip = $1 AND timestamp >= $2
		ORDER BY timestamp DESC
	`

	rows, err := DB.Query(query, ip, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query SMTP failures: %w", err)
	}
	defer rows.Close()

	var failures []SMTPFailure
	for rows.Next() {
		var f SMTPFailure
		err := rows.Scan(
			&f.ID, &f.SendingIP, &f.RecipientEmail, &f.RecipientDomain,
			&f.SMTPCode, &f.EnhancedCode, &f.Reason, &f.MXServer,
			&f.Timestamp, &f.EventID, &f.AttemptNumber,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan SMTP failure: %w", err)
		}
		failures = append(failures, f)
	}

	return failures, rows.Err()
}

// UpsertIPReputationMetrics inserts or updates IP reputation metrics
func UpsertIPReputationMetrics(metrics *IPReputationMetrics) error {
	// Marshal JSON fields
	reasonsJSON, err := json.Marshal(metrics.DistinctRejectionReasons)
	if err != nil {
		return fmt.Errorf("failed to marshal rejection reasons: %w", err)
	}

	providersJSON, err := json.Marshal(metrics.MajorProvidersRejecting)
	if err != nil {
		return fmt.Errorf("failed to marshal major providers: %w", err)
	}

	metadataJSON, err := json.Marshal(metrics.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO ip_reputation_metrics (
			ip, window_start, window_end, total_sent, total_rejected,
			rejection_ratio, unique_domains_rejected, distinct_rejection_reasons,
			major_providers_rejecting, status, last_updated, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (ip) DO UPDATE SET
			window_start = EXCLUDED.window_start,
			window_end = EXCLUDED.window_end,
			total_sent = EXCLUDED.total_sent,
			total_rejected = EXCLUDED.total_rejected,
			rejection_ratio = EXCLUDED.rejection_ratio,
			unique_domains_rejected = EXCLUDED.unique_domains_rejected,
			distinct_rejection_reasons = EXCLUDED.distinct_rejection_reasons,
			major_providers_rejecting = EXCLUDED.major_providers_rejecting,
			status = EXCLUDED.status,
			last_updated = EXCLUDED.last_updated,
			metadata = EXCLUDED.metadata
		RETURNING id
	`

	err = DB.QueryRow(
		query,
		metrics.IP,
		metrics.WindowStart,
		metrics.WindowEnd,
		metrics.TotalSent,
		metrics.TotalRejected,
		metrics.RejectionRatio,
		metrics.UniqueDomainsRejected,
		reasonsJSON,
		providersJSON,
		metrics.Status,
		metrics.LastUpdated,
		metadataJSON,
	).Scan(&metrics.ID)

	if err != nil {
		return fmt.Errorf("failed to upsert IP reputation metrics: %w", err)
	}

	return nil
}

// GetIPReputationMetrics retrieves reputation metrics for a specific IP
func GetIPReputationMetrics(ip string) (*IPReputationMetrics, error) {
	query := `
		SELECT id, ip, window_start, window_end, total_sent, total_rejected,
		       rejection_ratio, unique_domains_rejected, distinct_rejection_reasons,
		       major_providers_rejecting, status, last_updated, metadata
		FROM ip_reputation_metrics
		WHERE ip = $1
	`

	var metrics IPReputationMetrics
	var reasonsJSON, providersJSON, metadataJSON []byte

	err := DB.QueryRow(query, ip).Scan(
		&metrics.ID,
		&metrics.IP,
		&metrics.WindowStart,
		&metrics.WindowEnd,
		&metrics.TotalSent,
		&metrics.TotalRejected,
		&metrics.RejectionRatio,
		&metrics.UniqueDomainsRejected,
		&reasonsJSON,
		&providersJSON,
		&metrics.Status,
		&metrics.LastUpdated,
		&metadataJSON,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("IP reputation metrics not found")
		}
		return nil, fmt.Errorf("failed to get IP reputation metrics: %w", err)
	}

	// Unmarshal JSON fields
	if err := json.Unmarshal(reasonsJSON, &metrics.DistinctRejectionReasons); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rejection reasons: %w", err)
	}

	if err := json.Unmarshal(providersJSON, &metrics.MajorProvidersRejecting); err != nil {
		return nil, fmt.Errorf("failed to unmarshal major providers: %w", err)
	}

	if err := json.Unmarshal(metadataJSON, &metrics.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &metrics, nil
}

// GetAllIPReputationMetrics retrieves all IP reputation metrics with optional status filter
func GetAllIPReputationMetrics(status string) ([]IPReputationMetrics, error) {
	var query string
	var args []interface{}

	if status != "" {
		query = `
			SELECT id, ip, window_start, window_end, total_sent, total_rejected,
			       rejection_ratio, unique_domains_rejected, distinct_rejection_reasons,
			       major_providers_rejecting, status, last_updated, metadata
			FROM ip_reputation_metrics
			WHERE status = $1
			ORDER BY last_updated DESC
		`
		args = append(args, status)
	} else {
		query = `
			SELECT id, ip, window_start, window_end, total_sent, total_rejected,
			       rejection_ratio, unique_domains_rejected, distinct_rejection_reasons,
			       major_providers_rejecting, status, last_updated, metadata
			FROM ip_reputation_metrics
			ORDER BY last_updated DESC
		`
	}

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query IP reputation metrics: %w", err)
	}
	defer rows.Close()

	var metricsList []IPReputationMetrics
	for rows.Next() {
		var metrics IPReputationMetrics
		var reasonsJSON, providersJSON, metadataJSON []byte

		err := rows.Scan(
			&metrics.ID,
			&metrics.IP,
			&metrics.WindowStart,
			&metrics.WindowEnd,
			&metrics.TotalSent,
			&metrics.TotalRejected,
			&metrics.RejectionRatio,
			&metrics.UniqueDomainsRejected,
			&reasonsJSON,
			&providersJSON,
			&metrics.Status,
			&metrics.LastUpdated,
			&metadataJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan IP reputation metrics: %w", err)
		}

		// Unmarshal JSON fields
		if err := json.Unmarshal(reasonsJSON, &metrics.DistinctRejectionReasons); err != nil {
			return nil, fmt.Errorf("failed to unmarshal rejection reasons: %w", err)
		}

		if err := json.Unmarshal(providersJSON, &metrics.MajorProvidersRejecting); err != nil {
			return nil, fmt.Errorf("failed to unmarshal major providers: %w", err)
		}

		if err := json.Unmarshal(metadataJSON, &metrics.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		metricsList = append(metricsList, metrics)
	}

	return metricsList, rows.Err()
}

// InsertDNSBLCheck inserts a new DNSBL check result
func InsertDNSBLCheck(check *DNSBLCheck) error {
	listingsJSON, err := json.Marshal(check.Listings)
	if err != nil {
		return fmt.Errorf("failed to marshal listings: %w", err)
	}

	metadataJSON, err := json.Marshal(check.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO dnsbl_checks (
			ip, checked_at, listed, listings, check_duration_ms, metadata
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	err = DB.QueryRow(
		query,
		check.IP,
		check.CheckedAt,
		check.Listed,
		listingsJSON,
		check.CheckDurationMS,
		metadataJSON,
	).Scan(&check.ID)

	if err != nil {
		return fmt.Errorf("failed to insert DNSBL check: %w", err)
	}

	return nil
}

// GetLatestDNSBLCheck retrieves the most recent DNSBL check for an IP
func GetLatestDNSBLCheck(ip string) (*DNSBLCheck, error) {
	query := `
		SELECT id, ip, checked_at, listed, listings, check_duration_ms, metadata
		FROM dnsbl_checks
		WHERE ip = $1
		ORDER BY checked_at DESC
		LIMIT 1
	`

	var check DNSBLCheck
	var listingsJSON, metadataJSON []byte

	err := DB.QueryRow(query, ip).Scan(
		&check.ID,
		&check.IP,
		&check.CheckedAt,
		&check.Listed,
		&listingsJSON,
		&check.CheckDurationMS,
		&metadataJSON,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("DNSBL check not found")
		}
		return nil, fmt.Errorf("failed to get DNSBL check: %w", err)
	}

	// Unmarshal JSON fields
	if err := json.Unmarshal(listingsJSON, &check.Listings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal listings: %w", err)
	}

	if err := json.Unmarshal(metadataJSON, &check.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &check, nil
}

// InsertIPAction records an action taken on an IP
func InsertIPAction(action *IPAction) error {
	metadataJSON, err := json.Marshal(action.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO ip_actions (
			ip, action, previous_status, new_status, reason, triggered_by, metadata, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	err = DB.QueryRow(
		query,
		action.IP,
		action.Action,
		action.PreviousStatus,
		action.NewStatus,
		action.Reason,
		action.TriggeredBy,
		metadataJSON,
		action.CreatedAt,
	).Scan(&action.ID)

	if err != nil {
		return fmt.Errorf("failed to insert IP action: %w", err)
	}

	return nil
}

// GetIPActions retrieves actions for a specific IP
func GetIPActions(ip string, limit int) ([]IPAction, error) {
	query := `
		SELECT id, ip, action, previous_status, new_status, reason, triggered_by, metadata, created_at
		FROM ip_actions
		WHERE ip = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := DB.Query(query, ip, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query IP actions: %w", err)
	}
	defer rows.Close()

	var actions []IPAction
	for rows.Next() {
		var action IPAction
		var metadataJSON []byte

		err := rows.Scan(
			&action.ID,
			&action.IP,
			&action.Action,
			&action.PreviousStatus,
			&action.NewStatus,
			&action.Reason,
			&action.TriggeredBy,
			&metadataJSON,
			&action.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan IP action: %w", err)
		}

		// Unmarshal metadata
		if err := json.Unmarshal(metadataJSON, &action.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		actions = append(actions, action)
	}

	return actions, rows.Err()
}

// GetIPsNeedingAggregation returns IPs that have recent failures but need metrics update
func GetIPsNeedingAggregation(since time.Time) ([]string, error) {
	query := `
		SELECT DISTINCT sending_ip
		FROM smtp_failures
		WHERE timestamp >= $1
	`

	rows, err := DB.Query(query, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query IPs needing aggregation: %w", err)
	}
	defer rows.Close()

	var ips []string
	for rows.Next() {
		var ip string
		if err := rows.Scan(&ip); err != nil {
			return nil, fmt.Errorf("failed to scan IP: %w", err)
		}
		ips = append(ips, ip)
	}

	return ips, rows.Err()
}

// CleanOldSMTPFailures deletes SMTP failures older than specified duration
func CleanOldSMTPFailures(olderThan time.Time) (int64, error) {
	query := `DELETE FROM smtp_failures WHERE timestamp < $1`
	result, err := DB.Exec(query, olderThan)
	if err != nil {
		return 0, fmt.Errorf("failed to clean old SMTP failures: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}

// ExtractDomain extracts the domain from an email address
func ExtractDomain(email string) string {
	for i := len(email) - 1; i >= 0; i-- {
		if email[i] == '@' {
			return email[i+1:]
		}
	}
	return email
}

// IsMajorProvider checks if a domain is a major email provider
func IsMajorProvider(domain string) bool {
	majorProviders := []string{
		"gmail.com",
		"googlemail.com",
		"outlook.com",
		"hotmail.com",
		"live.com",
		"yahoo.com",
		"ymail.com",
		"aol.com",
		"icloud.com",
		"me.com",
		"protonmail.com",
		"mail.com",
	}

	for _, provider := range majorProviders {
		if domain == provider {
			return true
		}
	}
	return false
}

