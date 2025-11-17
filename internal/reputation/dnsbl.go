package reputation

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"golang-backend-service/internal/database"
	"golang-backend-service/internal/logger"

	"github.com/sirupsen/logrus"
)

// Major DNSBL providers to check
var MajorDNSBLs = []string{
	"zen.spamhaus.org",      // Spamhaus composite (most comprehensive)
	"b.barracudacentral.org", // Barracuda
	"bl.spamcop.net",        // SpamCop
	"cbl.abuseat.org",       // Composite Blocking List
	"dnsbl.sorbs.net",       // SORBS
	"bl.spamcannibal.org",   // SpamCannibal
	"psbl.surriel.com",      // Passive Spam Block List
	"dnsbl-1.uceprotect.net", // UCEProtect Level 1
}

// DNSBLResult represents the result of a DNSBL check
type DNSBLResult struct {
	IP              string
	Listed          bool
	Listings        []string
	CheckDurationMS int
	CheckedAt       time.Time
	Error           error
}

// CheckDNSBL performs a comprehensive DNSBL check for an IP address
func CheckDNSBL(ip string, timeoutSeconds int) (*DNSBLResult, error) {
	start := time.Now()

	logger.WithFields(logrus.Fields{
		"action": "dnsbl_check_start",
		"ip":     ip,
	}).Info("Starting DNSBL check")

	// Validate IP address
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return nil, fmt.Errorf("invalid IP address: %s", ip)
	}

	// Reverse the IP for DNSBL lookup
	reversedIP := reverseIP(ip)

	// Check all DNSBLs concurrently
	listings := checkAllDNSBLs(reversedIP, timeoutSeconds)

	duration := time.Since(start)
	result := &DNSBLResult{
		IP:              ip,
		Listed:          len(listings) > 0,
		Listings:        listings,
		CheckDurationMS: int(duration.Milliseconds()),
		CheckedAt:       time.Now(),
	}

	// Record DNSBL check metric
	RecordDNSBLCheck(ip, result.Listed, duration.Seconds())

	// Store result in database
	dbCheck := &database.DNSBLCheck{
		IP:              result.IP,
		CheckedAt:       result.CheckedAt,
		Listed:          result.Listed,
		Listings:        result.Listings,
		CheckDurationMS: result.CheckDurationMS,
		Metadata:        make(map[string]interface{}),
	}

	if err := database.InsertDNSBLCheck(dbCheck); err != nil {
		logger.WithFields(logrus.Fields{
			"action": "dnsbl_check_store_failed",
			"ip":     ip,
			"error":  err.Error(),
		}).Error("Failed to store DNSBL check result")
	}

	logger.WithFields(logrus.Fields{
		"action":      "dnsbl_check_complete",
		"ip":          ip,
		"listed":      result.Listed,
		"listings":    len(listings),
		"duration_ms": result.CheckDurationMS,
	}).Info("DNSBL check completed")

	return result, nil
}

// checkAllDNSBLs checks an IP against all DNSBLs concurrently
func checkAllDNSBLs(reversedIP string, timeoutSeconds int) []string {
	var wg sync.WaitGroup
	var mu sync.Mutex
	listings := []string{}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	for _, dnsbl := range MajorDNSBLs {
		wg.Add(1)
		go func(dnsbl string) {
			defer wg.Done()

			query := fmt.Sprintf("%s.%s", reversedIP, dnsbl)
			
			// Create resolver with timeout
			resolver := &net.Resolver{
				PreferGo: true,
			}

			_, err := resolver.LookupHost(ctx, query)
			
			// If no error, the IP is listed
			if err == nil {
				mu.Lock()
				listings = append(listings, dnsbl)
				mu.Unlock()

				logger.WithFields(logrus.Fields{
					"action": "dnsbl_listing_found",
					"dnsbl":  dnsbl,
					"query":  query,
				}).Warn("IP found on DNSBL")
			}
			// NXDOMAIN or timeout means not listed (expected)
		}(dnsbl)
	}

	wg.Wait()
	return listings
}

// reverseIP reverses an IP address for DNSBL lookup
func reverseIP(ip string) string {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return ip
	}

	// Reverse the octets
	return fmt.Sprintf("%s.%s.%s.%s", parts[3], parts[2], parts[1], parts[0])
}

// CheckDNSBLAsync performs DNSBL check asynchronously
func CheckDNSBLAsync(ip string, timeoutSeconds int, callback func(*DNSBLResult, error)) {
	go func() {
		result, err := CheckDNSBL(ip, timeoutSeconds)
		callback(result, err)
	}()
}

// GetDNSBLSeverity returns severity level based on which DNSBLs list the IP
func GetDNSBLSeverity(listings []string) string {
	if len(listings) == 0 {
		return "none"
	}

	// Spamhaus is the most critical
	for _, listing := range listings {
		if strings.Contains(listing, "spamhaus") {
			return "critical"
		}
	}

	// Multiple listings is high severity
	if len(listings) >= 3 {
		return "high"
	}

	if len(listings) >= 2 {
		return "medium"
	}

	return "low"
}

// FormatDNSBLReport creates a human-readable DNSBL report
func FormatDNSBLReport(result *DNSBLResult) string {
	if !result.Listed {
		return fmt.Sprintf("✅ IP %s is NOT listed on any DNSBL (checked %d lists)", 
			result.IP, len(MajorDNSBLs))
	}

	severity := GetDNSBLSeverity(result.Listings)
	report := fmt.Sprintf("⚠️  IP %s IS LISTED on %d DNSBL(s) - Severity: %s\n\n", 
		result.IP, len(result.Listings), strings.ToUpper(severity))

	report += "Listed on:\n"
	for _, listing := range result.Listings {
		report += fmt.Sprintf("  - %s\n", listing)
	}

	report += fmt.Sprintf("\nCheck completed in %dms\n", result.CheckDurationMS)

	// Add remediation advice
	report += "\nRecommended Actions:\n"
	if severity == "critical" || severity == "high" {
		report += "  1. Immediately quarantine this IP\n"
		report += "  2. Investigate the root cause (compromised account, spam content)\n"
		report += "  3. Submit delisting requests after fixing issues\n"
		report += "  4. Switch to backup IP addresses\n"
	} else {
		report += "  1. Investigate recent sending patterns\n"
		report += "  2. Review email content and list quality\n"
		report += "  3. Monitor sending rates\n"
		report += "  4. Submit delisting request if appropriate\n"
	}

	return report
}

// BatchCheckDNSBL checks multiple IPs concurrently
func BatchCheckDNSBL(ips []string, timeoutSeconds int) map[string]*DNSBLResult {
	results := make(map[string]*DNSBLResult)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, ip := range ips {
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()
			
			result, err := CheckDNSBL(ip, timeoutSeconds)
			if err != nil {
				logger.WithFields(logrus.Fields{
					"action": "batch_dnsbl_check_error",
					"ip":     ip,
					"error":  err.Error(),
				}).Error("DNSBL check failed")
				return
			}

			mu.Lock()
			results[ip] = result
			mu.Unlock()
		}(ip)
	}

	wg.Wait()
	return results
}

// ShouldCheckDNSBL determines if an IP should be checked based on last check time
func ShouldCheckDNSBL(ip string, minHoursBetweenChecks int) (bool, error) {
	lastCheck, err := database.GetLatestDNSBLCheck(ip)
	if err != nil {
		// If not found, should check
		if strings.Contains(err.Error(), "not found") {
			return true, nil
		}
		return false, err
	}

	// Check if enough time has passed
	timeSinceLastCheck := time.Since(lastCheck.CheckedAt)
	minDuration := time.Duration(minHoursBetweenChecks) * time.Hour

	return timeSinceLastCheck >= minDuration, nil
}

// GetDelistingInstructions provides instructions for delisting from specific DNSBLs
func GetDelistingInstructions(dnsbl string) string {
	instructions := map[string]string{
		"zen.spamhaus.org": "Visit https://www.spamhaus.org/lookup/ and follow the delisting process. Requires identifying and fixing the root cause.",
		"b.barracudacentral.org": "Visit http://www.barracudacentral.org/rbl/removal-request to submit a removal request.",
		"bl.spamcop.net": "Visit https://www.spamcop.net/bl.shtml for delisting information. Listings auto-expire after 24 hours.",
		"cbl.abuseat.org": "Visit https://cbl.abuseat.org/lookup.cgi for lookup and delisting info. Usually indicates a compromised system.",
		"dnsbl.sorbs.net": "Visit http://www.sorbs.net/delisting/ for delisting procedures. May require investigation.",
	}

	if instr, exists := instructions[dnsbl]; exists {
		return instr
	}

	return "Search for '" + dnsbl + " delisting' for removal instructions specific to this DNSBL."
}

