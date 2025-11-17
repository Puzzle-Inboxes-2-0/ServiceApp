package reputation

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Prometheus metrics for IP reputation system
var (
	// Counter for SMTP failures processed
	SMTPFailuresTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "smtp_failures_total",
			Help: "Total number of SMTP failures processed",
		},
		[]string{"ip", "enhanced_code", "domain"},
	)

	// Counter for IP status changes
	IPStatusChangesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ip_status_changes_total",
			Help: "Total number of IP status changes",
		},
		[]string{"ip", "from_status", "to_status"},
	)

	// Gauge for current IP statuses
	IPStatusGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ip_reputation_status",
			Help: "Current IP reputation status (1=healthy, 2=warning, 3=quarantine, 4=blacklisted)",
		},
		[]string{"ip"},
	)

	// Histogram for rejection ratios
	RejectionRatioHistogram = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "ip_rejection_ratio",
			Help:    "Distribution of IP rejection ratios",
			Buckets: []float64{0.01, 0.02, 0.03, 0.05, 0.10, 0.20, 0.50},
		},
	)

	// Counter for DNSBL checks
	DNSBLChecksTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "dnsbl_checks_total",
			Help: "Total number of DNSBL checks performed",
		},
		[]string{"ip", "listed"},
	)

	// Histogram for DNSBL check duration
	DNSBLCheckDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "dnsbl_check_duration_seconds",
			Help:    "Duration of DNSBL checks in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	// Counter for aggregation runs
	AggregationRunsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ip_aggregation_runs_total",
			Help: "Total number of IP aggregation runs",
		},
		[]string{"status"},
	)

	// Gauge for IPs processed in last aggregation
	IPsProcessedGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "ips_processed_last_run",
			Help: "Number of IPs processed in last aggregation run",
		},
	)

	// Counter for webhook events received
	WebhookEventsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "webhook_events_total",
			Help: "Total number of webhook events received",
		},
		[]string{"event_type", "status"},
	)
)

// GetStatusValue converts status string to numeric value for metrics
func GetStatusValue(status string) float64 {
	switch status {
	case "healthy":
		return 1
	case "warning":
		return 2
	case "quarantine":
		return 3
	case "blacklisted":
		return 4
	default:
		return 0
	}
}

// RecordSMTPFailure records an SMTP failure metric
func RecordSMTPFailure(ip, enhancedCode, domain string) {
	SMTPFailuresTotal.WithLabelValues(ip, enhancedCode, domain).Inc()
}

// RecordStatusChange records an IP status change metric
func RecordStatusChange(ip, fromStatus, toStatus string) {
	IPStatusChangesTotal.WithLabelValues(ip, fromStatus, toStatus).Inc()
	IPStatusGauge.WithLabelValues(ip).Set(GetStatusValue(toStatus))
}

// RecordRejectionRatio records a rejection ratio observation
func RecordRejectionRatio(ratio float64) {
	RejectionRatioHistogram.Observe(ratio)
}

// RecordDNSBLCheck records a DNSBL check metric
func RecordDNSBLCheck(ip string, listed bool, durationSeconds float64) {
	listedStr := "false"
	if listed {
		listedStr = "true"
	}
	DNSBLChecksTotal.WithLabelValues(ip, listedStr).Inc()
	DNSBLCheckDuration.Observe(durationSeconds)
}

// RecordAggregationRun records an aggregation run metric
func RecordAggregationRun(status string, ipsProcessed int) {
	AggregationRunsTotal.WithLabelValues(status).Inc()
	IPsProcessedGauge.Set(float64(ipsProcessed))
}

// RecordWebhookEvent records a webhook event metric
func RecordWebhookEvent(eventType, status string) {
	WebhookEventsTotal.WithLabelValues(eventType, status).Inc()
}

