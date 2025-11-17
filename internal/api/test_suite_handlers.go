package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"golang-backend-service/internal/database"
	"golang-backend-service/internal/logger"
	"golang-backend-service/internal/reputation"

	"github.com/sirupsen/logrus"
)

// TestCase represents a single test scenario
type TestCase struct {
	ID              string                   `json:"id"`
	Name            string                   `json:"name"`
	Description     string                   `json:"description"`
	IP              string                   `json:"ip"`
	TotalSent       int                      `json:"total_sent"`
	Failures        []FailureSimulation      `json:"failures"`
	ExpectedStatus  string                   `json:"expected_status"`
	Category        string                   `json:"category"`
}

// TestResult represents the result of running a test case
type TestResult struct {
	TestID          string    `json:"test_id"`
	TestName        string    `json:"test_name"`
	ExpectedStatus  string    `json:"expected_status"`
	ActualStatus    string    `json:"actual_status"`
	Passed          bool      `json:"passed"`
	ExecutionTime   float64   `json:"execution_time_ms"`
	Timestamp       time.Time `json:"timestamp"`
	ErrorMessage    string    `json:"error_message,omitempty"`
	RejectionRatio  float64   `json:"rejection_ratio"`
	FailureCount    int       `json:"failure_count"`
}

// TestSuiteResult represents the results of running all tests
type TestSuiteResult struct {
	TotalTests      int          `json:"total_tests"`
	PassedTests     int          `json:"passed_tests"`
	FailedTests     int          `json:"failed_tests"`
	ExecutionTime   float64      `json:"execution_time_ms"`
	Timestamp       time.Time    `json:"timestamp"`
	Results         []TestResult `json:"results"`
}

// GetTestCases returns all predefined test cases
func getTestCases() []TestCase {
	return []TestCase{
		{
			ID:          "test-1",
			Name:        "Healthy IP - Normal Operations",
			Description: "IP with minimal failures - should remain healthy",
			IP:          "203.0.113.10",
			TotalSent:   500,
			Failures: []FailureSimulation{
				{Code: "5.1.1", Domain: "unknown-domain.com", Count: 1, Reason: "Recipient not found"},
				{Code: "4.2.2", Domain: "example.com", Count: 1, Reason: "Mailbox full"},
			},
			ExpectedStatus: "healthy",
			Category:       "normal",
		},
		{
			ID:          "test-2",
			Name:        "Warning State - Elevated Rejections",
			Description: "IP with elevated rejection rate from major providers",
			IP:          "203.0.113.11",
			TotalSent:   300,
			Failures: []FailureSimulation{
				{Code: "5.7.1", Domain: "gmail.com", Count: 3, Reason: "IP reputation"},
				{Code: "5.7.1", Domain: "outlook.com", Count: 2, Reason: "Policy reject"},
				{Code: "5.1.1", Domain: "various.com", Count: 3, Reason: "Unknown user"},
			},
			ExpectedStatus: "warning",
			Category:       "escalation",
		},
		{
			ID:          "test-3",
			Name:        "Quarantine - Multiple Major Providers Rejecting",
			Description: "Multiple major providers rejecting - should quarantine",
			IP:          "203.0.113.12",
			TotalSent:   400,
			Failures: []FailureSimulation{
				{Code: "5.7.1", Domain: "gmail.com", Count: 7, Reason: "IP reputation"},
				{Code: "5.7.1", Domain: "outlook.com", Count: 5, Reason: "Policy reject"},
				{Code: "4.7.0", Domain: "yahoo.com", Count: 3, Reason: "Temporarily deferred"},
			},
			ExpectedStatus: "quarantine",
			Category:       "escalation",
		},
		{
			ID:          "test-4",
			Name:        "Blacklisted - Critical Reputation Damage",
			Description: "High rejection rate across multiple major providers",
			IP:          "203.0.113.13",
			TotalSent:   500,
			Failures: []FailureSimulation{
				{Code: "5.7.1", Domain: "gmail.com", Count: 12, Reason: "IP reputation"},
				{Code: "5.7.1", Domain: "outlook.com", Count: 10, Reason: "Blocked by policy"},
				{Code: "5.7.1", Domain: "yahoo.com", Count: 8, Reason: "Spam detected"},
				{Code: "5.7.1", Domain: "aol.com", Count: 5, Reason: "IP on blocklist"},
			},
			ExpectedStatus: "blacklisted",
			Category:       "critical",
		},
		{
			ID:          "test-5",
			Name:        "Low Volume - Insufficient Data",
			Description: "Low volume with failures - should stay healthy due to insufficient data",
			IP:          "203.0.113.14",
			TotalSent:   20,
			Failures: []FailureSimulation{
				{Code: "5.7.1", Domain: "gmail.com", Count: 2, Reason: "IP reputation"},
				{Code: "5.1.1", Domain: "example.com", Count: 1, Reason: "Unknown user"},
			},
			ExpectedStatus: "healthy",
			Category:       "edge-case",
		},
		{
			ID:          "test-6",
			Name:        "Temporary Throttling - 4xx Codes",
			Description: "Mostly temporary failures - should trigger warning",
			IP:          "203.0.113.15",
			TotalSent:   600,
			Failures: []FailureSimulation{
				{Code: "4.7.0", Domain: "gmail.com", Count: 12, Reason: "Rate limited"},
				{Code: "4.2.1", Domain: "outlook.com", Count: 4, Reason: "Mailbox busy"},
				{Code: "5.7.1", Domain: "yahoo.com", Count: 2, Reason: "Policy"},
			},
			ExpectedStatus: "warning",
			Category:       "throttling",
		},
		{
			ID:          "test-7",
			Name:        "SPF/DKIM Failures - Configuration Issue",
			Description: "Authentication failures - should quarantine for investigation",
			IP:          "203.0.113.16",
			TotalSent:   300,
			Failures: []FailureSimulation{
				{Code: "5.7.23", Domain: "gmail.com", Count: 15, Reason: "SPF validation failed"},
				{Code: "5.7.1", Domain: "outlook.com", Count: 10, Reason: "DKIM fail"},
			},
			ExpectedStatus: "quarantine",
			Category:       "configuration",
		},
		{
			ID:          "test-8",
			Name:        "PTR Record Missing - DNS Issue",
			Description: "Reverse DNS failures - should quarantine",
			IP:          "203.0.113.17",
			TotalSent:   200,
			Failures: []FailureSimulation{
				{Code: "5.7.25", Domain: "gmail.com", Count: 8, Reason: "PTR record required"},
				{Code: "5.7.25", Domain: "outlook.com", Count: 4, Reason: "Reverse DNS lookup failed"},
			},
			ExpectedStatus: "quarantine",
			Category:       "configuration",
		},
		{
			ID:          "test-9",
			Name:        "Mixed Signals - Hard to Classify",
			Description: "Mixed error types - should trigger warning",
			IP:          "203.0.113.18",
			TotalSent:   450,
			Failures: []FailureSimulation{
				{Code: "5.1.1", Domain: "example1.com", Count: 5, Reason: "Unknown user"},
				{Code: "5.7.1", Domain: "gmail.com", Count: 3, Reason: "Policy"},
				{Code: "4.2.2", Domain: "example2.com", Count: 3, Reason: "Mailbox full"},
			},
			ExpectedStatus: "warning",
			Category:       "mixed",
		},
		{
			ID:          "test-10",
			Name:        "Gradual Decay - Early Stage",
			Description: "Initial healthy state with minimal failures",
			IP:          "203.0.113.19",
			TotalSent:   300,
			Failures: []FailureSimulation{
				{Code: "5.1.1", Domain: "example.com", Count: 3, Reason: "Unknown user"},
			},
			ExpectedStatus: "healthy",
			Category:       "progression",
		},
		{
			ID:          "test-11",
			Name:        "Microsoft Reputation Block - 5.7.606",
			Description: "Microsoft-specific access denied - critical reputation damage",
			IP:          "203.0.113.20",
			TotalSent:   400,
			Failures: []FailureSimulation{
				{Code: "5.7.606", Domain: "outlook.com", Count: 8, Reason: "Access denied, bad reputation"},
				{Code: "5.7.606", Domain: "hotmail.com", Count: 6, Reason: "Sender blocked"},
				{Code: "5.7.1", Domain: "live.com", Count: 4, Reason: "Policy block"},
			},
			ExpectedStatus: "quarantine",
			Category:       "critical",
		},
		{
			ID:          "test-12",
			Name:        "Content Spam Detection - 5.7.512",
			Description: "Message content rejected as spam - critical issue",
			IP:          "203.0.113.21",
			TotalSent:   350,
			Failures: []FailureSimulation{
				{Code: "5.7.512", Domain: "gmail.com", Count: 5, Reason: "Message content rejected"},
				{Code: "5.7.512", Domain: "outlook.com", Count: 4, Reason: "Spam detected"},
				{Code: "5.7.1", Domain: "yahoo.com", Count: 3, Reason: "Content policy violation"},
			},
			ExpectedStatus: "quarantine",
			Category:       "critical",
		},
		{
			ID:          "test-13",
			Name:        "Infrastructure Issues - Multiple DNS Problems",
			Description: "MX/DNS/PTR combined infrastructure failures",
			IP:          "203.0.113.22",
			TotalSent:   250,
			Failures: []FailureSimulation{
				{Code: "5.7.27", Domain: "enterprise.com", Count: 5, Reason: "Sender address has null MX"},
				{Code: "5.7.7", Domain: "business.net", Count: 4, Reason: "Domain has no MX record"},
				{Code: "5.1.8", Domain: "corporate.org", Count: 4, Reason: "Bad sender's system address"},
			},
			ExpectedStatus: "quarantine",
			Category:       "configuration",
		},
		{
			ID:          "test-14",
			Name:        "DKIM/ARC Authentication Failure - 5.7.26",
			Description: "Sender authentication required (ARC/DKIM failures)",
			IP:          "203.0.113.23",
			TotalSent:   300,
			Failures: []FailureSimulation{
				{Code: "5.7.26", Domain: "gmail.com", Count: 12, Reason: "ARC validation failed"},
				{Code: "5.7.26", Domain: "yahoo.com", Count: 8, Reason: "DKIM signature required"},
			},
			ExpectedStatus: "quarantine",
			Category:       "configuration",
		},
		{
			ID:          "test-15",
			Name:        "Policy Rejections - Temporary Issues",
			Description: "Mixed temporary policy rejections and recipient issues",
			IP:          "203.0.113.24",
			TotalSent:   500,
			Failures: []FailureSimulation{
				{Code: "4.7.1", Domain: "gmail.com", Count: 8, Reason: "Temporary policy rejection"},
				{Code: "5.7.510", Domain: "outlook.com", Count: 6, Reason: "Recipient address rejected"},
				{Code: "5.4.1", Domain: "yahoo.com", Count: 4, Reason: "Recipient address no longer available"},
			},
			ExpectedStatus: "warning",
			Category:       "policy",
		},
	}
}

// @Summary Get all test cases
// @Description Retrieve all predefined IP reputation test scenarios
// @Tags testing
// @Produce json
// @Success 200 {array} TestCase
// @Router /api/testing/test-cases [get]
func getTestCasesHandler(w http.ResponseWriter, r *http.Request) {
	testCases := getTestCases()
	
	logger.WithFields(logrus.Fields{
		"action":     "get_test_cases",
		"test_count": len(testCases),
	}).Info("Retrieved test cases")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(testCases)
}

// @Summary Run a single test case
// @Description Execute a single test case by ID and return the result
// @Tags testing
// @Produce json
// @Param id path string true "Test Case ID (e.g., test-1)"
// @Success 200 {object} TestResult
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/testing/test-cases/{id}/run [post]
func runTestCaseHandler(w http.ResponseWriter, r *http.Request) {
	// Extract test ID from URL
	testID := r.URL.Query().Get("id")
	if testID == "" {
		// Try getting from path (for Swagger compatibility)
		testID = r.PathValue("id")
	}

	logger.WithFields(logrus.Fields{
		"action":  "run_test_case",
		"test_id": testID,
	}).Info("Running test case")

	// Find the test case
	testCases := getTestCases()
	var testCase *TestCase
	for i := range testCases {
		if testCases[i].ID == testID {
			testCase = &testCases[i]
			break
		}
	}

	if testCase == nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "test_not_found",
			Message: "Test case not found: " + testID,
		})
		return
	}

	// Run the test
	startTime := time.Now()
	result := executeTestCase(testCase)
	result.ExecutionTime = float64(time.Since(startTime).Microseconds()) / 1000.0

	logger.WithFields(logrus.Fields{
		"action":        "test_case_completed",
		"test_id":       testID,
		"passed":        result.Passed,
		"expected":      result.ExpectedStatus,
		"actual":        result.ActualStatus,
		"execution_ms":  result.ExecutionTime,
	}).Info("Test case completed")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// @Summary Run all test cases
// @Description Execute all predefined test cases and return comprehensive results
// @Tags testing
// @Produce json
// @Success 200 {object} TestSuiteResult
// @Failure 500 {object} ErrorResponse
// @Router /api/testing/test-suite/run [post]
func runTestSuiteHandler(w http.ResponseWriter, r *http.Request) {
	logger.Info("Starting test suite execution")
	
	startTime := time.Now()
	testCases := getTestCases()
	results := make([]TestResult, 0, len(testCases))
	passedCount := 0
	failedCount := 0

	for _, testCase := range testCases {
		tcStartTime := time.Now()
		result := executeTestCase(&testCase)
		result.ExecutionTime = float64(time.Since(tcStartTime).Microseconds()) / 1000.0
		
		results = append(results, result)
		if result.Passed {
			passedCount++
		} else {
			failedCount++
		}
	}

	suite := TestSuiteResult{
		TotalTests:    len(testCases),
		PassedTests:   passedCount,
		FailedTests:   failedCount,
		ExecutionTime: float64(time.Since(startTime).Microseconds()) / 1000.0,
		Timestamp:     time.Now(),
		Results:       results,
	}

	logger.WithFields(logrus.Fields{
		"action":       "test_suite_completed",
		"total":        suite.TotalTests,
		"passed":       suite.PassedTests,
		"failed":       suite.FailedTests,
		"execution_ms": suite.ExecutionTime,
	}).Info("Test suite completed")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(suite)
}

// executeTestCase runs a single test case and returns the result
func executeTestCase(testCase *TestCase) TestResult {
	result := TestResult{
		TestID:         testCase.ID,
		TestName:       testCase.Name,
		ExpectedStatus: testCase.ExpectedStatus,
		Timestamp:      time.Now(),
	}

	// Simulate the failures using the existing endpoint logic
	payload := FailureSimulationPayload{
		IP:        testCase.IP,
		TotalSent: testCase.TotalSent,
		Failures:  testCase.Failures,
	}

	// Insert failures
	totalFailures := 0
	for _, failure := range payload.Failures {
		for i := 0; i < failure.Count; i++ {
			smtpFailure := &database.SMTPFailure{
				SendingIP:       payload.IP,
				RecipientEmail:  "test@" + failure.Domain,
				RecipientDomain: failure.Domain,
				SMTPCode:        parseEnhancedCodeToSMTP(failure.Code),
				EnhancedCode:    failure.Code,
				Reason:          failure.Reason,
				MXServer:        "mx." + failure.Domain,
				Timestamp:       time.Now().Add(-time.Duration(i) * time.Minute),
				EventID:         "test-" + testCase.ID + "-" + strconv.Itoa(i),
				AttemptNumber:   1,
			}

			if err := database.InsertSMTPFailure(smtpFailure); err != nil {
				result.ErrorMessage = "Failed to insert failure: " + err.Error()
				result.Passed = false
				return result
			}
			totalFailures++
		}
	}

	result.FailureCount = totalFailures

	// Calculate rejection ratio
	if payload.TotalSent > 0 {
		result.RejectionRatio = float64(totalFailures) / float64(payload.TotalSent)
	}

	// Trigger aggregation
	config := reputation.DefaultReputationConfig()
	metrics, err := reputation.AggregateIPOnDemand(payload.IP, config)
	if err != nil {
		result.ErrorMessage = "Failed to aggregate: " + err.Error()
		result.Passed = false
		return result
	}

	result.ActualStatus = metrics.Status

	// Check if test passed
	result.Passed = (result.ActualStatus == result.ExpectedStatus)

	return result
}

// Helper functions
func parseEnhancedCodeToSMTP(enhancedCode string) int {
	if len(enhancedCode) > 0 {
		switch enhancedCode[0] {
		case '2':
			return 200
		case '4':
			return 400
		case '5':
			return 500
		}
	}
	return 550
}

