package reputation

import (
	"fmt"
	"strings"
	"time"

	"golang-backend-service/internal/database"
)

// Configuration for reputation thresholds
type ReputationConfig struct {
	WindowMinutes                  int     `json:"window_minutes"`
	MinVolumeForAssessment         int     `json:"min_volume_for_assessment"`
	BlacklistRejectionRatio        float64 `json:"blacklist_rejection_ratio"`
	BlacklistMinDomains            int     `json:"blacklist_min_domains"`
	BlacklistMinMajorProviders     int     `json:"blacklist_min_major_providers"`
	QuarantineRejectionRatio       float64 `json:"quarantine_rejection_ratio"`
	QuarantineMinDomains           int     `json:"quarantine_min_domains"`
	WarningRejectionRatio          float64 `json:"warning_rejection_ratio"`
	WarningReputationCodeThreshold int     `json:"warning_reputation_code_threshold"`
}

// DefaultReputationConfig returns the default configuration
func DefaultReputationConfig() ReputationConfig {
	return ReputationConfig{
		WindowMinutes:                  15,
		MinVolumeForAssessment:         50,
		BlacklistRejectionRatio:        0.05, // 5%
		BlacklistMinDomains:            3,
		BlacklistMinMajorProviders:     2,
		QuarantineRejectionRatio:       0.03, // 3%
		QuarantineMinDomains:           2,
		WarningRejectionRatio:          0.02, // 2%
		WarningReputationCodeThreshold: 5,
	}
}

// IPHealthCheck contains calculated metrics for IP health assessment
type IPHealthCheck struct {
	IP                    string         `json:"ip"`
	WindowMinutes         int            `json:"window_minutes"`
	TotalSent             int            `json:"total_sent"`
	TotalRejected         int            `json:"total_rejected"`
	RejectionRatio        float64        `json:"rejection_ratio"`
	UniqueDomainsRejected int            `json:"unique_domains_rejected"`
	MajorProviders        []string       `json:"major_providers"`
	ReputationCodes       map[string]int `json:"reputation_codes"`
	ThrottleCount         int            `json:"throttle_count"`
	DomainCounts          map[string]int `json:"domain_counts"`
}

// DetermineIPStatus applies the decision algorithm to determine IP status
func DetermineIPStatus(metrics IPHealthCheck, config ReputationConfig) string {
	// CRITICAL: Must have minimum volume to assess
	if metrics.TotalSent < config.MinVolumeForAssessment {
		return "healthy"
	}

	// BLACKLISTED - Immediate action required
	if isBlacklisted(metrics, config) {
		return "blacklisted"
	}

	// QUARANTINE - High risk, needs investigation
	if isQuarantined(metrics, config) {
		return "quarantine"
	}

	// WARNING - Monitor closely
	if isWarning(metrics, config) {
		return "warning"
	}

	return "healthy"
}

// isBlacklisted checks if IP meets blacklist criteria
func isBlacklisted(metrics IPHealthCheck, config ReputationConfig) bool {
	return metrics.RejectionRatio > config.BlacklistRejectionRatio &&
		metrics.UniqueDomainsRejected >= config.BlacklistMinDomains &&
		len(metrics.MajorProviders) >= config.BlacklistMinMajorProviders &&
		hasReputationRelatedCodes(metrics.ReputationCodes)
}

// isQuarantined checks if IP meets quarantine criteria
func isQuarantined(metrics IPHealthCheck, config ReputationConfig) bool {
	// High rejection rate with at least one major provider
	if metrics.RejectionRatio > config.QuarantineRejectionRatio && len(metrics.MajorProviders) >= 1 {
		return true
	}

	// 5%+ rejection with 2+ domains
	if metrics.RejectionRatio > config.BlacklistRejectionRatio && metrics.UniqueDomainsRejected >= config.QuarantineMinDomains {
		return true
	}

	return false
}

// isWarning checks if IP meets warning criteria
func isWarning(metrics IPHealthCheck, config ReputationConfig) bool {
	// 2%+ rejection rate sustained
	if metrics.RejectionRatio >= config.WarningRejectionRatio {
		return true
	}

	// Many 4xx codes plus some 5xx
	if metrics.ThrottleCount > 10 && metrics.TotalRejected > 0 {
		return true
	}

	// Repeated 5.7.1 patterns
	if hasRepeated571Patterns(metrics.ReputationCodes) {
		return true
	}

	return false
}

// hasReputationRelatedCodes checks if reputation-related codes are present
func hasReputationRelatedCodes(codes map[string]int) bool {
	// PRIMARY reputation codes (most severe)
	primaryCodes := []string{
		"5.7.1",   // IP/domain reputation blocked
		"5.7.606", // Access denied (Microsoft-specific)
		"5.7.512", // Message content rejected (spam)
	}

	// AUTHENTICATION codes (configuration issues that affect reputation)
	authCodes := []string{
		"5.7.23", // SPF validation failed
		"5.7.26", // Sender authentication required (ARC/DKIM)
	}

	// INFRASTRUCTURE codes (DNS/PTR issues)
	infraCodes := []string{
		"5.7.25", // PTR record required
		"5.7.27", // Sender address has null MX
		"5.7.7",  // Domain name has no MX/A/AAAA record
		"5.1.8",  // Bad sender's system address
	}

	// POLICY codes (temporary rejections)
	policyCodes := []string{
		"4.7.0",   // Temporary rate limit/greylisting
		"4.7.1",   // Temporary policy rejection
		"5.7.510", // Recipient address rejected (policy)
	}

	// Check primary codes (most critical - lower threshold)
	for _, code := range primaryCodes {
		if count, exists := codes[code]; exists && count >= 2 {
			return true
		}
	}

	// Check authentication codes
	for _, code := range authCodes {
		if count, exists := codes[code]; exists && count >= 3 {
			return true
		}
	}

	// Check infrastructure codes
	for _, code := range infraCodes {
		if count, exists := codes[code]; exists && count >= 3 {
			return true
		}
	}

	// Check policy codes (higher threshold as these can be temporary)
	for _, code := range policyCodes {
		if count, exists := codes[code]; exists && count >= 5 {
			return true
		}
	}

	return false
}

// hasRepeated571Patterns checks for repeated 5.7.1 codes
func hasRepeated571Patterns(codes map[string]int) bool {
	if count, exists := codes["5.7.1"]; exists && count >= 5 {
		return true
	}
	return false
}

// CalculateIPHealthCheck computes health metrics from SMTP failures
func CalculateIPHealthCheck(ip string, windowMinutes int, totalSent int) (*IPHealthCheck, error) {
	windowStart := time.Now().Add(-time.Duration(windowMinutes) * time.Minute)

	// Get all failures in the window
	failures, err := database.GetSMTPFailuresByIP(ip, windowStart)
	if err != nil {
		return nil, fmt.Errorf("failed to get SMTP failures: %w", err)
	}

	health := &IPHealthCheck{
		IP:                    ip,
		WindowMinutes:         windowMinutes,
		TotalSent:             totalSent,
		TotalRejected:         len(failures),
		RejectionRatio:        0.0,
		UniqueDomainsRejected: 0,
		MajorProviders:        []string{},
		ReputationCodes:       make(map[string]int),
		ThrottleCount:         0,
		DomainCounts:          make(map[string]int),
	}

	// Calculate rejection ratio
	if totalSent > 0 {
		health.RejectionRatio = float64(health.TotalRejected) / float64(totalSent)
	}

	// Analyze failures
	domainSet := make(map[string]bool)
	majorProviderSet := make(map[string]bool)

	for _, failure := range failures {
		// Count domains
		domainSet[failure.RecipientDomain] = true
		health.DomainCounts[failure.RecipientDomain]++

		// Count enhanced codes
		if failure.EnhancedCode != "" {
			health.ReputationCodes[failure.EnhancedCode]++

			// Count 4xx throttles
			if strings.HasPrefix(failure.EnhancedCode, "4") {
				health.ThrottleCount++
			}
		}

		// Track major providers
		if database.IsMajorProvider(failure.RecipientDomain) {
			majorProviderSet[failure.RecipientDomain] = true
		}
	}

	// Set unique counts
	health.UniqueDomainsRejected = len(domainSet)

	// Convert major providers set to slice
	for provider := range majorProviderSet {
		health.MajorProviders = append(health.MajorProviders, provider)
	}

	return health, nil
}

// GetStatusSummary provides a human-readable summary of the IP status determination
func GetStatusSummary(status string, health IPHealthCheck) string {
	switch status {
	case "blacklisted":
		return fmt.Sprintf(
			"CRITICAL: IP %s is BLACKLISTED. Rejection ratio: %.2f%%, %d unique domains rejected, %d major providers rejecting. Immediate action required.",
			health.IP,
			health.RejectionRatio*100,
			health.UniqueDomainsRejected,
			len(health.MajorProviders),
		)
	case "quarantine":
		return fmt.Sprintf(
			"WARNING: IP %s is QUARANTINED. Rejection ratio: %.2f%%, %d unique domains rejected. High risk, needs investigation.",
			health.IP,
			health.RejectionRatio*100,
			health.UniqueDomainsRejected,
		)
	case "warning":
		return fmt.Sprintf(
			"CAUTION: IP %s has WARNING status. Rejection ratio: %.2f%%. Monitor closely.",
			health.IP,
			health.RejectionRatio*100,
		)
	default:
		return fmt.Sprintf(
			"OK: IP %s is HEALTHY. Rejection ratio: %.2f%%.",
			health.IP,
			health.RejectionRatio*100,
		)
	}
}

// GetRecommendedActions returns recommended actions based on status
func GetRecommendedActions(status string) []string {
	switch status {
	case "blacklisted":
		return []string{
			"immediate_quarantine",
			"swap_to_backup_ip",
			"run_dnsbl_checks",
			"alert_ops_critical",
			"investigate_root_cause",
		}
	case "quarantine":
		return []string{
			"reduce_traffic_50_percent",
			"run_dnsbl_checks",
			"alert_ops_warning",
			"monitor_closely",
		}
	case "warning":
		return []string{
			"monitor_closely",
			"reduce_send_rate",
			"check_email_list_hygiene",
		}
	default:
		return []string{"continue_normal_operations"}
	}
}

// IsReputationIssue determines if the failures are IP reputation related
func IsReputationIssue(health IPHealthCheck) bool {
	// Check for reputation-specific error codes
	reputationCodes := []string{"5.7.1", "5.7.25", "5.7.23"}

	for _, code := range reputationCodes {
		if count, exists := health.ReputationCodes[code]; exists && count > 0 {
			return true
		}
	}

	// If most errors are 5.1.1 (unknown user), it's likely a list hygiene issue
	if count511, exists := health.ReputationCodes["5.1.1"]; exists {
		if float64(count511)/float64(health.TotalRejected) > 0.7 {
			return false // Not reputation, likely bad email list
		}
	}

	return false
}

// GetIssueType categorizes the type of issue based on error codes
func GetIssueType(health IPHealthCheck) string {
	// SPAM DETECTION - Most severe
	spamCodes := []string{"5.7.512", "5.7.606"}
	for _, code := range spamCodes {
		if count, exists := health.ReputationCodes[code]; exists && count > 2 {
			return "content_spam_detected"
		}
	}

	// IP REPUTATION - Direct reputation damage
	if count, exists := health.ReputationCodes["5.7.1"]; exists && count > 5 {
		return "ip_reputation_damage"
	}

	// AUTHENTICATION failures (SPF/DKIM/DMARC/ARC)
	authCodes := []string{"5.7.23", "5.7.26"}
	totalAuthFailures := 0
	for _, code := range authCodes {
		if count, exists := health.ReputationCodes[code]; exists {
			totalAuthFailures += count
		}
	}
	if totalAuthFailures > 5 {
		return "authentication_failure"
	}

	// INFRASTRUCTURE issues (DNS/PTR/MX)
	infraCodes := []string{"5.7.25", "5.7.27", "5.7.7", "5.1.8"}
	totalInfraFailures := 0
	for _, code := range infraCodes {
		if count, exists := health.ReputationCodes[code]; exists {
			totalInfraFailures += count
		}
	}
	if totalInfraFailures > 5 {
		return "infrastructure_misconfiguration"
	}

	// POLICY rejections
	policyCodes := []string{"5.7.510", "4.7.1"}
	totalPolicyFailures := 0
	for _, code := range policyCodes {
		if count, exists := health.ReputationCodes[code]; exists {
			totalPolicyFailures += count
		}
	}
	if totalPolicyFailures > 10 {
		return "policy_violation"
	}

	// Unknown users (list hygiene)
	if count, exists := health.ReputationCodes["5.1.1"]; exists && count > 10 {
		return "list_hygiene_issue"
	}

	// Throttling/rate limiting
	if health.ThrottleCount > health.TotalRejected/2 {
		return "rate_limiting"
	}

	return "mixed_issues"
}
