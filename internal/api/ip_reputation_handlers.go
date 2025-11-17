package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang-backend-service/internal/database"
	"golang-backend-service/internal/logger"
	"golang-backend-service/internal/reputation"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// WebhookEvent represents a Stalwart webhook event
type WebhookEvent struct {
	ID        string    `json:"id"`
	CreatedAt string    `json:"createdAt"`
	Type      string    `json:"type"`
	Data      EventData `json:"data"`
}

// EventData represents the data payload in a webhook event
type EventData struct {
	Domain        string `json:"domain"`
	Recipient     string `json:"recipient"`
	IP            string `json:"ip"`
	SMTPCode      int    `json:"smtp_code"`
	EnhancedCode  string `json:"enhanced_code"`
	Reason        string `json:"reason"`
	MX            string `json:"mx"`
	AttemptNumber int    `json:"attempt_number"`
}

// WebhookPayload represents the complete webhook payload from Stalwart
type WebhookPayload struct {
	Events []WebhookEvent `json:"events"`
}

// IPReputationResponse represents the API response for IP reputation
type IPReputationResponse struct {
	IP              string                        `json:"ip"`
	Status          string                        `json:"status"`
	Metrics         *database.IPReputationMetrics `json:"metrics"`
	LatestDNSBL     *database.DNSBLCheck          `json:"latest_dnsbl_check"`
	RecentActions   []database.IPAction           `json:"recent_actions"`
	Summary         string                        `json:"summary"`
	Recommendations []string                      `json:"recommendations"`
}

// IPHealthDashboardResponse represents dashboard data
type IPHealthDashboardResponse struct {
	Timestamp      time.Time                      `json:"timestamp"`
	TotalIPs       int                            `json:"total_ips"`
	HealthyIPs     int                            `json:"healthy_ips"`
	WarningIPs     int                            `json:"warning_ips"`
	QuarantineIPs  int                            `json:"quarantine_ips"`
	BlacklistedIPs int                            `json:"blacklisted_ips"`
	IPDetails      []database.IPReputationMetrics `json:"ip_details"`
}

// FailureSimulation represents a simulated SMTP failure for testing
type FailureSimulation struct {
	Code   string `json:"code"`
	Domain string `json:"domain"`
	Count  int    `json:"count"`
	Reason string `json:"reason"`
}

// FailureSimulationPayload represents the payload for simulating failures
type FailureSimulationPayload struct {
	IP        string              `json:"ip"`
	TotalSent int                 `json:"total_sent"`
	Failures  []FailureSimulation `json:"failures"`
}

// @Summary Process Stalwart delivery failure webhook
// @Description Receive and process SMTP delivery failure events from Stalwart
// @Tags webhooks
// @Accept json
// @Produce json
// @Param payload body WebhookPayload true "Webhook payload"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/webhooks/stalwart/delivery-failure [post]
func processDeliveryFailureHandler(w http.ResponseWriter, r *http.Request) {
	var payload WebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		logger.WithFields(logrus.Fields{
			"action": "webhook_decode_failed",
			"error":  err.Error(),
		}).Error("Failed to decode webhook payload")

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "invalid_payload",
			Message: "Failed to decode webhook payload",
		})
		return
	}

	processedCount := 0
	failedCount := 0

	for _, event := range payload.Events {
		// Only process delivery failure events
		if event.Type != "smtp.delivery.failure" {
			continue
		}

		// Extract domain from recipient email
		domain := database.ExtractDomain(event.Data.Recipient)

		// Create SMTP failure record
		failure := &database.SMTPFailure{
			SendingIP:       event.Data.IP,
			RecipientEmail:  event.Data.Recipient,
			RecipientDomain: domain,
			SMTPCode:        event.Data.SMTPCode,
			EnhancedCode:    event.Data.EnhancedCode,
			Reason:          event.Data.Reason,
			MXServer:        event.Data.MX,
			Timestamp:       time.Now(),
			EventID:         event.ID,
			AttemptNumber:   event.Data.AttemptNumber,
		}

		// Parse timestamp if provided
		if event.CreatedAt != "" {
			if parsed, err := time.Parse(time.RFC3339, event.CreatedAt); err == nil {
				failure.Timestamp = parsed
			}
		}

		// Insert failure record
		if err := database.InsertSMTPFailure(failure); err != nil {
			logger.WithFields(logrus.Fields{
				"action":   "insert_failure_failed",
				"event_id": event.ID,
				"ip":       event.Data.IP,
				"error":    err.Error(),
			}).Error("Failed to insert SMTP failure")
			reputation.RecordWebhookEvent(event.Type, "failed")
			failedCount++
			continue
		}

		// Record metrics
		reputation.RecordSMTPFailure(event.Data.IP, event.Data.EnhancedCode, domain)
		reputation.RecordWebhookEvent(event.Type, "success")

		processedCount++

		logger.WithFields(logrus.Fields{
			"action":        "smtp_failure_recorded",
			"event_id":      event.ID,
			"ip":            event.Data.IP,
			"recipient":     event.Data.Recipient,
			"smtp_code":     event.Data.SMTPCode,
			"enhanced_code": event.Data.EnhancedCode,
		}).Info("SMTP failure recorded")
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "success",
		"processed": processedCount,
		"failed":    failedCount,
		"total":     len(payload.Events),
	})
}

// @Summary Get IP reputation
// @Description Retrieve reputation metrics and status for a specific IP
// @Tags ip-reputation
// @Produce json
// @Param ip path string true "IP Address"
// @Success 200 {object} IPReputationResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/ips/{ip}/reputation [get]
func getIPReputationHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ip := vars["ip"]

	// Get reputation metrics
	metrics, err := database.GetIPReputationMetrics(ip)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error:   "not_found",
				Message: "No reputation data found for this IP",
			})
			return
		}

		logger.WithFields(logrus.Fields{
			"action": "get_reputation_failed",
			"ip":     ip,
			"error":  err.Error(),
		}).Error("Failed to get IP reputation")

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "database_error",
			Message: "Failed to retrieve IP reputation",
		})
		return
	}

	// Get latest DNSBL check (optional)
	latestDNSBL, _ := database.GetLatestDNSBLCheck(ip)

	// Get recent actions
	recentActions, _ := database.GetIPActions(ip, 10)

	// Calculate health for summary
	config := reputation.DefaultReputationConfig()
	health, _ := reputation.CalculateIPHealthCheck(ip, config.WindowMinutes, metrics.TotalSent)

	response := IPReputationResponse{
		IP:              ip,
		Status:          metrics.Status,
		Metrics:         metrics,
		LatestDNSBL:     latestDNSBL,
		RecentActions:   recentActions,
		Summary:         reputation.GetStatusSummary(metrics.Status, *health),
		Recommendations: reputation.GetRecommendedActions(metrics.Status),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// @Summary Get SMTP failures for IP
// @Description Retrieve SMTP failures for a specific IP within a time window
// @Tags ip-reputation
// @Produce json
// @Param ip path string true "IP Address"
// @Param window query string false "Time window (e.g., 15m, 1h, 24h)" default(15m)
// @Success 200 {array} database.SMTPFailure
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/ips/{ip}/failures [get]
func getIPFailuresHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ip := vars["ip"]

	// Parse window parameter
	windowStr := r.URL.Query().Get("window")
	if windowStr == "" {
		windowStr = "15m"
	}

	duration, err := time.ParseDuration(windowStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "invalid_window",
			Message: "Invalid time window format (use 15m, 1h, 24h, etc.)",
		})
		return
	}

	since := time.Now().Add(-duration)

	// Get failures
	failures, err := database.GetSMTPFailuresByIP(ip, since)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"action": "get_failures_failed",
			"ip":     ip,
			"error":  err.Error(),
		}).Error("Failed to get SMTP failures")

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "database_error",
			Message: "Failed to retrieve SMTP failures",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(failures)
}

// @Summary Manually quarantine IP
// @Description Manually set an IP to quarantine status
// @Tags ip-reputation
// @Produce json
// @Param ip path string true "IP Address"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} ErrorResponse
// @Router /api/ips/{ip}/quarantine [post]
func quarantineIPHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ip := vars["ip"]

	// Trigger on-demand aggregation with quarantine override
	config := reputation.DefaultReputationConfig()
	metrics, err := reputation.AggregateIPOnDemand(ip, config)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"action": "quarantine_failed",
			"ip":     ip,
			"error":  err.Error(),
		}).Error("Failed to quarantine IP")

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "quarantine_failed",
			Message: "Failed to quarantine IP",
		})
		return
	}

	// Force quarantine status
	metrics.Status = "quarantine"
	if err := database.UpsertIPReputationMetrics(metrics); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "database_error",
			Message: "Failed to update IP status",
		})
		return
	}

	// Record manual action
	action := &database.IPAction{
		IP:          ip,
		Action:      "manual_quarantine",
		NewStatus:   "quarantine",
		Reason:      "Manually quarantined via API",
		TriggeredBy: "manual",
		Metadata:    make(map[string]interface{}),
		CreatedAt:   time.Now(),
	}
	database.InsertIPAction(action)

	// Trigger DNSBL check
	go reputation.CheckDNSBL(ip, 5)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"ip":      ip,
		"message": "IP has been quarantined",
	})
}

// @Summary Check DNSBL status for IP
// @Description Run DNSBL checks for a specific IP
// @Tags ip-reputation
// @Produce json
// @Param ip path string true "IP Address"
// @Success 200 {object} reputation.DNSBLResult
// @Failure 500 {object} ErrorResponse
// @Router /api/ips/{ip}/dnsbl-check [post]
func checkDNSBLHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ip := vars["ip"]

	// Run DNSBL check
	result, err := reputation.CheckDNSBL(ip, 5)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"action": "dnsbl_check_failed",
			"ip":     ip,
			"error":  err.Error(),
		}).Error("DNSBL check failed")

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "dnsbl_check_failed",
			Message: "Failed to perform DNSBL check",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// @Summary Get IP health dashboard
// @Description Retrieve aggregated IP health metrics for dashboard
// @Tags ip-reputation
// @Produce json
// @Param status query string false "Filter by status (healthy, warning, quarantine, blacklisted)"
// @Success 200 {object} IPHealthDashboardResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/dashboard/ip-health [get]
func getIPHealthDashboardHandler(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")

	// Get all IP metrics (optionally filtered by status)
	allMetrics, err := database.GetAllIPReputationMetrics(status)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"action": "get_dashboard_failed",
			"error":  err.Error(),
		}).Error("Failed to get dashboard data")

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "database_error",
			Message: "Failed to retrieve dashboard data",
		})
		return
	}

	// Count by status
	statusCounts := map[string]int{
		"healthy":     0,
		"warning":     0,
		"quarantine":  0,
		"blacklisted": 0,
	}

	for _, m := range allMetrics {
		statusCounts[m.Status]++
	}

	response := IPHealthDashboardResponse{
		Timestamp:      time.Now(),
		TotalIPs:       len(allMetrics),
		HealthyIPs:     statusCounts["healthy"],
		WarningIPs:     statusCounts["warning"],
		QuarantineIPs:  statusCounts["quarantine"],
		BlacklistedIPs: statusCounts["blacklisted"],
		IPDetails:      allMetrics,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// @Summary Simulate SMTP failures for testing
// @Description Simulate SMTP failures for testing the reputation system
// @Tags testing
// @Accept json
// @Produce json
// @Param test_data body FailureSimulationPayload true "Test scenario data"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/testing/simulate-failures [post]
func simulateFailuresHandler(w http.ResponseWriter, r *http.Request) {
	var testData FailureSimulationPayload
	if err := json.NewDecoder(r.Body).Decode(&testData); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid test data",
		})
		return
	}

	// Validate test parameters
	if testData.IP == "" || testData.TotalSent == 0 || len(testData.Failures) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "invalid_test_data",
			Message: "Required fields: ip, total_sent, failures",
		})
		return
	}

	// Insert test failures
	insertedCount := 0
	for _, failure := range testData.Failures {
		// Insert multiple failures based on count
		for i := 0; i < failure.Count; i++ {
			smtpFailure := &database.SMTPFailure{
				SendingIP:       testData.IP,
				RecipientEmail:  "test@" + failure.Domain,
				RecipientDomain: failure.Domain,
				SMTPCode:        550,
				EnhancedCode:    failure.Code,
				Reason:          failure.Reason,
				MXServer:        "mx." + failure.Domain,
				Timestamp:       time.Now().Add(-time.Duration(i) * time.Minute),
				EventID:         "test-" + strconv.Itoa(int(time.Now().Unix())) + "-" + strconv.Itoa(i),
				AttemptNumber:   1,
			}

			if err := database.InsertSMTPFailure(smtpFailure); err == nil {
				insertedCount++
			}
		}
	}

	// Trigger aggregation
	config := reputation.DefaultReputationConfig()
	metrics, err := reputation.AggregateIPOnDemand(testData.IP, config)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "aggregation_failed",
			Message: "Failed to aggregate test data",
		})
		return
	}

	// Get health summary
	health, _ := reputation.CalculateIPHealthCheck(testData.IP, config.WindowMinutes, testData.TotalSent)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":           "success",
		"failures_created": insertedCount,
		"ip_status":        metrics.Status,
		"metrics":          metrics,
		"summary":          reputation.GetStatusSummary(metrics.Status, *health),
		"recommendations":  reputation.GetRecommendedActions(metrics.Status),
	})
}
