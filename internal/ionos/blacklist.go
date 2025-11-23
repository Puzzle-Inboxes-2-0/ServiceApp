package ionos

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// DNSBLChecker checks IPs against DNS-based blacklists
type DNSBLChecker struct {
	blacklists []string
	ignored    []string
	timeout    time.Duration
	logger     *logrus.Logger
}

// NewDNSBLChecker creates a new DNSBL checker
func NewDNSBLChecker(logger *logrus.Logger) *DNSBLChecker {
	// Comprehensive list of DNSBLs (excluding UCEPROTECT and Invaluement per user requirements)
	blacklists := []string{
		// Spamhaus
		"zen.spamhaus.org",
		// Barracuda
		"b.barracudacentral.org",
		// Spamcop
		"bl.spamcop.net",
		// Abuseat
		"cbl.abuseat.org",
		// SpamRats
		"dyna.spamrats.com",
		"noptr.spamrats.com",
		"spam.spamrats.com",
		// Manitu (NiX Spam)
		"ix.dnsbl.manitu.net",
		// SORBS
		"dnsbl.sorbs.net",
		// SURRIEL
		"psbl.surriel.com",
		// UBL
		"ubl.unsubscore.com",
		// DroneBL
		"dnsbl.dronebl.org",
	}

	// Blacklists to IGNORE (user accepts these as clean OR they give false positives)
	ignored := []string{
		"dnsbl-1.uceprotect.net",
		"dnsbl-2.uceprotect.net",
		"dnsbl-3.uceprotect.net",
		"sip.invaluement.com",
		"sip24.invaluement.com",
	}

	return &DNSBLChecker{
		blacklists: blacklists,
		ignored:    ignored,
		timeout:    2 * time.Second,
		logger:     logger,
	}
}

// BlacklistResult contains the result of a blacklist check
type BlacklistResult struct {
	IP            string
	IsBlacklisted bool
	Blacklists    []string
	CheckDuration time.Duration
}

// CheckIP checks if an IP is blacklisted
func (c *DNSBLChecker) CheckIP(ctx context.Context, ip string) (*BlacklistResult, error) {
	startTime := time.Now()

	// Reverse the IP address for DNS query (e.g. 1.2.3.4 -> 4.3.2.1)
	reversedIP, err := reverseIP(ip)
	if err != nil {
		return nil, fmt.Errorf("failed to reverse IP: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"action":      "check_blacklist",
		"ip":          ip,
		"reversed_ip": reversedIP,
	}).Info("Checking IP against blacklists")

	// Use a WaitGroup to check all blacklists concurrently
	var wg sync.WaitGroup
	blacklistsChan := make(chan string, len(c.blacklists))
	
	for _, bl := range c.blacklists {
		wg.Add(1)
		go func(blacklist string) {
			defer wg.Done()
			if c.checkSingleBlacklist(ctx, reversedIP, blacklist) {
				blacklistsChan <- blacklist
			}
		}(bl)
	}

	// Close channel when all goroutines complete
	go func() {
		wg.Wait()
		close(blacklistsChan)
	}()

	// Collect results
	foundBlacklists := []string{}
	for bl := range blacklistsChan {
		// Filter out ignored blacklists
		if !c.isIgnored(bl) {
			foundBlacklists = append(foundBlacklists, bl)
		}
	}

	duration := time.Since(startTime)
	result := &BlacklistResult{
		IP:            ip,
		IsBlacklisted: len(foundBlacklists) > 0,
		Blacklists:    foundBlacklists,
		CheckDuration: duration,
	}

	c.logger.WithFields(logrus.Fields{
		"action":        "check_blacklist",
		"ip":            ip,
		"is_blacklisted": result.IsBlacklisted,
		"blacklists":    result.Blacklists,
		"duration_ms":   duration.Milliseconds(),
	}).Info("Blacklist check completed")

	return result, nil
}

// checkSingleBlacklist checks a single blacklist
func (c *DNSBLChecker) checkSingleBlacklist(ctx context.Context, reversedIP, blacklist string) bool {
	query := fmt.Sprintf("%s.%s", reversedIP, blacklist)
	
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: c.timeout,
			}
			return d.DialContext(ctx, network, address)
		},
	}

	// Create a context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err := resolver.LookupHost(timeoutCtx, query)
	
	// If resolution succeeds, the IP is listed
	if err == nil {
		return true
	}

	// Check if it's a timeout or other error
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		c.logger.WithFields(logrus.Fields{
			"action":    "check_single_blacklist",
			"blacklist": blacklist,
			"query":     query,
		}).Warn("Blacklist check timed out")
	}

	return false
}

// isIgnored checks if a blacklist should be ignored
func (c *DNSBLChecker) isIgnored(blacklist string) bool {
	for _, ignored := range c.ignored {
		if strings.Contains(blacklist, ignored) {
			return true
		}
	}
	return false
}

// reverseIP reverses an IP address for DNSBL queries
func reverseIP(ip string) (string, error) {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return "", fmt.Errorf("invalid IP address format: %s", ip)
	}

	// Reverse the parts
	reversed := make([]string, 4)
	for i := 0; i < 4; i++ {
		reversed[i] = parts[3-i]
	}

	return strings.Join(reversed, "."), nil
}

