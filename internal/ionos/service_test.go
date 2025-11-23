package ionos

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
)

// TestDNSBLChecker tests the blacklist checker
func TestDNSBLChecker(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce noise during tests
	
	checker := NewDNSBLChecker(logger)
	
	tests := []struct {
		name          string
		ip            string
		shouldBeClean bool
	}{
		{
			name:          "Known clean IP - Google DNS",
			ip:            "8.8.8.8",
			shouldBeClean: true,
		},
		{
			name:          "Known bad IP - TEST-NET-1",
			ip:            "192.0.2.1",
			shouldBeClean: true, // TEST-NET addresses are usually not listed
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := checker.CheckIP(ctx, tt.ip)
			
			if err != nil {
				t.Fatalf("CheckIP failed: %v", err)
			}
			
			if result.IP != tt.ip {
				t.Errorf("Expected IP %s, got %s", tt.ip, result.IP)
			}
			
			// Note: We can't guarantee the blacklist status of any IP in tests
			// as it depends on external DNSBL services
			t.Logf("IP %s - Blacklisted: %v, Blacklists: %v, Duration: %v",
				result.IP, result.IsBlacklisted, result.Blacklists, result.CheckDuration)
		})
	}
}

// TestReverseIP tests IP address reversal
func TestReverseIP(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected string
		wantErr  bool
	}{
		{
			name:     "Valid IP",
			ip:       "1.2.3.4",
			expected: "4.3.2.1",
			wantErr:  false,
		},
		{
			name:     "Another valid IP",
			ip:       "192.168.1.1",
			expected: "1.1.168.192",
			wantErr:  false,
		},
		{
			name:     "Invalid IP - too few octets",
			ip:       "1.2.3",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "Invalid IP - too many octets",
			ip:       "1.2.3.4.5",
			expected: "",
			wantErr:  true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := reverseIP(tt.ip)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestIgnoredBlacklists tests that certain blacklists are ignored
func TestIgnoredBlacklists(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	checker := NewDNSBLChecker(logger)
	
	ignoredTests := []string{
		"dnsbl-1.uceprotect.net",
		"dnsbl-2.uceprotect.net",
		"dnsbl-3.uceprotect.net",
		"sip.invaluement.com",
		"sip24.invaluement.com",
	}
	
	for _, bl := range ignoredTests {
		t.Run("Should ignore "+bl, func(t *testing.T) {
			isIgnored := checker.isIgnored(bl)
			if !isIgnored {
				t.Errorf("Blacklist %s should be ignored but is not", bl)
			}
		})
	}
}

// TestBlacklistConfiguration tests that the expected blacklists are configured
func TestBlacklistConfiguration(t *testing.T) {
	logger := logrus.New()
	checker := NewDNSBLChecker(logger)
	
	expectedBlacklists := []string{
		"zen.spamhaus.org",
		"b.barracudacentral.org",
		"bl.spamcop.net",
		"cbl.abuseat.org",
		"dyna.spamrats.com",
		"noptr.spamrats.com",
		"spam.spamrats.com",
		"ix.dnsbl.manitu.net",
		"dnsbl.sorbs.net",
		"psbl.surriel.com",
		"ubl.unsubscore.com",
		"dnsbl.dronebl.org",
	}
	
	if len(checker.blacklists) != len(expectedBlacklists) {
		t.Errorf("Expected %d blacklists, got %d", len(expectedBlacklists), len(checker.blacklists))
	}
	
	blacklistMap := make(map[string]bool)
	for _, bl := range checker.blacklists {
		blacklistMap[bl] = true
	}
	
	for _, expected := range expectedBlacklists {
		if !blacklistMap[expected] {
			t.Errorf("Expected blacklist %s not found in configuration", expected)
		}
	}
}

