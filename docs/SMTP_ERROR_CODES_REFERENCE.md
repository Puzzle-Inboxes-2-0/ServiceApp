# SMTP Error Codes Reference & Processing Guide

**Complete reference for IP reputation system error code handling**

Last Updated: November 16, 2025

---

## Table of Contents

1. [API Call Structure](#api-call-structure)
2. [Complete Error Code Coverage](#complete-error-code-coverage)
3. [Data Processing Flow](#data-processing-flow)
4. [Storage Architecture](#storage-architecture)
5. [Decision Algorithm](#decision-algorithm)
6. [Architectural Consequences](#architectural-consequences)

---

## API Call Structure

### 1. Webhook Endpoint (Primary Entry Point)

**POST** `/api/webhooks/stalwart/delivery-failure`

This is where Stalwart (or any SMTP server) sends delivery failure events.

#### Request Structure

```json
{
  "events": [
    {
      "id": "unique-event-id-123",
      "createdAt": "2025-11-16T10:30:00Z",
      "type": "smtp.delivery.failure",
      "data": {
        "ip": "203.0.113.10",
        "recipient": "user@example.com",
        "smtp_code": 550,
        "enhanced_code": "5.7.1",
        "reason": "Message rejected due to IP reputation",
        "mx": "mx.example.com",
        "attempt_number": 1
      }
    }
  ]
}
```

#### Field Descriptions

| Field | Type | Description | Required |
|-------|------|-------------|----------|
| `id` | string | Unique event identifier | Yes |
| `createdAt` | ISO8601 | Event timestamp | Yes |
| `type` | string | Must be "smtp.delivery.failure" | Yes |
| `data.ip` | string | Sending IP address (IPv4/IPv6) | Yes |
| `data.recipient` | email | Recipient email address | Yes |
| `data.smtp_code` | int | SMTP response code (e.g., 550) | Yes |
| `data.enhanced_code` | string | Enhanced status code (e.g., "5.7.1") | Yes |
| `data.reason` | string | Human-readable failure reason | Yes |
| `data.mx` | string | Receiving MX server hostname | Yes |
| `data.attempt_number` | int | Delivery attempt number | Yes |

#### Response

```json
{
  "status": "success",
  "processed": 1,
  "failed": 0,
  "total": 1
}
```

### 2. Reputation Query Endpoint

**GET** `/api/ips/{ip}/reputation`

Returns comprehensive reputation data for an IP.

#### Response Structure

```json
{
  "ip": "203.0.113.10",
  "status": "warning",
  "metrics": {
    "ip": "203.0.113.10",
    "window_start": "2025-11-16T10:00:00Z",
    "window_end": "2025-11-16T10:15:00Z",
    "total_sent": 500,
    "total_rejected": 15,
    "rejection_ratio": 0.03,
    "unique_domains_rejected": 3,
    "distinct_rejection_reasons": {
      "5.7.1": 10,
      "5.7.23": 3,
      "5.1.1": 2
    },
    "major_providers_rejecting": ["gmail.com", "outlook.com"],
    "status": "warning",
    "last_updated": "2025-11-16T10:15:00Z"
  },
  "latest_dnsbl_check": {
    "ip": "203.0.113.10",
    "checked_at": "2025-11-16T10:14:00Z",
    "listed": false,
    "listings": [],
    "check_duration_ms": 1234
  },
  "recent_actions": [...],
  "summary": "CAUTION: IP 203.0.113.10 has WARNING status. Rejection ratio: 3.00%. Monitor closely.",
  "recommendations": [
    "monitor_closely",
    "reduce_send_rate",
    "check_email_list_hygiene"
  ]
}
```

### 3. Other API Endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/ips/{ip}/failures?window=15m` | GET | Get SMTP failures for IP |
| `/api/ips/{ip}/quarantine` | POST | Manually quarantine IP |
| `/api/ips/{ip}/dnsbl-check` | POST | Run DNSBL check |
| `/api/dashboard/ip-health` | GET | Dashboard data for all IPs |
| `/api/testing/simulate-failures` | POST | Simulate failures (testing) |

---

## Complete Error Code Coverage

### Primary Reputation Codes (CRITICAL)

**Threshold: 2+ occurrences triggers reputation concern**

| Code | Meaning | Impact | Action Required |
|------|---------|--------|-----------------|
| **5.7.1** | IP/domain reputation blocked | CRITICAL | Immediate investigation, DNSBL check |
| **5.7.606** | Access denied (Microsoft-specific) | CRITICAL | Check Microsoft SNDS, investigate spam |
| **5.7.512** | Message content rejected (spam) | CRITICAL | Review content, check for compromise |

**Architectural Decision**: These codes are given a **lower threshold (2 occurrences)** because they directly indicate reputation damage. By the time you see 3+ of these, deliverability is already severely impacted.

**2nd Order Consequence**: Early detection means faster response time, potentially preventing full blacklisting.

**3rd Order Consequence**: Reduces false positives from transient issues while catching real problems early.

### Authentication Codes (HIGH PRIORITY)

**Threshold: 3+ occurrences**

| Code | Meaning | Root Cause | Fix |
|------|---------|------------|-----|
| **5.7.23** | SPF validation failed | SPF record incorrect or missing | Update DNS SPF record |
| **5.7.26** | Authentication required (ARC/DKIM) | DKIM signature invalid or missing | Fix DKIM signing, check private keys |

**Architectural Decision**: Authentication failures don't immediately indicate reputation damage, but they will lead to reputation damage over time as providers lose trust.

**2nd Order Consequence**: These are often configuration issues that can be fixed quickly if detected early.

**3rd Order Consequence**: If ignored, leads to gradual reputation decay across multiple providers.

### Infrastructure Codes (MEDIUM PRIORITY)

**Threshold: 3+ occurrences**

| Code | Meaning | Root Cause | Fix |
|------|---------|------------|-----|
| **5.7.25** | PTR record required | No reverse DNS configured | Add PTR record in DNS |
| **5.7.27** | Sender address has null MX | Sender domain has no MX record | Configure MX records |
| **5.7.7** | Domain has no MX/A/AAAA | DNS misconfiguration | Fix DNS records |
| **5.1.8** | Bad sender's system address | Invalid sender address format | Fix email address formatting |

**Architectural Decision**: Infrastructure issues are serious but often affect specific providers. They indicate setup problems rather than reputation damage.

**2nd Order Consequence**: These problems prevent emails from being accepted before reputation evaluation even occurs.

**3rd Order Consequence**: Can mask actual reputation issues - fix these first to see true reputation status.

### Policy & Temporary Codes

**Threshold: 5+ occurrences**

| Code | Meaning | Severity | Notes |
|------|---------|----------|-------|
| **4.7.0** | Temporary rate limit/greylisting | LOW | Often temporary, retry mechanisms handle |
| **4.7.1** | Temporary policy rejection | LOW | May be greylisting or temporary rate limit |
| **5.7.510** | Recipient address rejected (policy) | MEDIUM | Policy violation, review sending patterns |

**Architectural Decision**: Higher threshold (5 occurrences) because temporary issues (4xx codes) are expected and shouldn't trigger immediate alerts.

**2nd Order Consequence**: Prevents alert fatigue from transient network issues.

**3rd Order Consequence**: If these persist, they indicate sending too fast - need to implement backoff.

### List Hygiene Codes

| Code | Meaning | Impact |
|------|---------|--------|
| **5.1.1** | Recipient not found / Unknown user | Indicates poor list quality, not reputation issue |

**Architectural Decision**: This is tracked but handled separately because it indicates list hygiene problems, not IP reputation issues.

**Critical 4th Order Consequence**: High 5.1.1 rates can eventually damage reputation if providers see you as not managing your lists properly.

---

## Data Processing Flow

### Phase 1: Webhook Reception

```
Stalwart/SMTP Server → Webhook Handler → Validation → Database Insert
                                            ↓
                                    Prometheus Metrics
```

**Processing Steps:**

1. **Receive webhook** at `/api/webhooks/stalwart/delivery-failure`
2. **Parse JSON** and validate structure
3. **Extract domain** from recipient email
4. **Create SMTPFailure record**:
   ```go
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
   ```
5. **Insert to database** (`smtp_failures` table)
6. **Record Prometheus metrics**:
   - `smtp_failures_total{ip, enhanced_code, domain}`
   - `webhook_events_total{event_type, status}`

**Architectural Consequences:**

- **Synchronous Processing**: Webhook handler blocks until DB insert completes
  - **Pro**: Guaranteed persistence before ack
  - **Con**: Slow DB could cause webhook timeouts
  - **Mitigation**: Connection pooling (25 max connections)

- **No Deduplication**: Same event_id can be inserted multiple times
  - **Risk**: Stalwart retry logic could cause duplicates
  - **4th Order Consequence**: Inflated failure counts leading to false positives
  - **Recommended Fix**: Add UNIQUE constraint on event_id field

### Phase 2: Background Aggregation

**Runs every 5 minutes automatically**

```
Timer Trigger → Get IPs with Recent Failures → For Each IP:
                                                   ↓
                                          Calculate Health Metrics
                                                   ↓
                                          Determine Status
                                                   ↓
                                          Update Database
                                                   ↓
                                          Trigger Actions (DNSBL, alerts)
```

**Aggregation Steps:**

1. **Identify IPs** needing aggregation (have failures in last 15 minutes)
2. **Calculate metrics** for each IP:
   ```go
   health := CalculateIPHealthCheck(ip, 15_minutes, total_sent)
   // Returns: rejection_ratio, unique_domains, error_code_counts, etc.
   ```
3. **Determine status** using decision algorithm:
   ```go
   status := DetermineIPStatus(health, config)
   // Returns: "healthy" | "warning" | "quarantine" | "blacklisted"
   ```
4. **Update `ip_reputation_metrics`** table
5. **If status changed**:
   - Record action in `ip_actions` table
   - Update Prometheus gauge
   - Trigger DNSBL check (async)
   - Log alert

**Architectural Consequences:**

- **15-Minute Window**: All calculations use rolling 15-minute window
  - **Pro**: Fast reaction to problems
  - **Con**: Vulnerable to burst traffic causing false positives
  - **3rd Order**: Could trigger unnecessary IP swaps, wasting clean IPs

- **Estimated Total Sent**: Currently estimates based on failures
  - **Risk**: Estimation formula assumes 5% failure rate baseline
  - **4th Order Consequence**: If actual baseline is different, all thresholds are wrong
  - **Critical Fix Needed**: Integrate with actual sending metrics from Stalwart

### Phase 3: Decision Algorithm

**Multi-tiered evaluation with priority ordering**

```go
if total_sent < 50 → return "healthy"  // Insufficient volume
↓
if BLACKLISTED_criteria → return "blacklisted"
↓
if QUARANTINE_criteria → return "quarantine"
↓
if WARNING_criteria → return "warning"
↓
return "healthy"
```

**Blacklist Criteria (ALL must be true):**
- Rejection ratio > 5%
- 3+ unique domains rejecting
- 2+ major providers (Gmail, Outlook, Yahoo, etc.)
- Reputation-related error codes present

**2nd Order Consequence**: Very strict criteria means fewer false positives but potentially slower detection.

**3rd Order Consequence**: By the time all criteria are met, reputation damage is already significant.

**5th Order Consequence**: Industry reputation systems (like Gmail's) might have already downgraded you before you detect it internally.

---

## Storage Architecture

### Table: smtp_failures

**Purpose**: Raw event storage for all delivery failures

```sql
CREATE TABLE smtp_failures (
    id SERIAL PRIMARY KEY,
    sending_ip VARCHAR(45) NOT NULL,
    recipient_email VARCHAR(255) NOT NULL,
    recipient_domain VARCHAR(255) NOT NULL,
    smtp_code INTEGER,
    enhanced_code VARCHAR(10),
    reason TEXT,
    mx_server VARCHAR(255),
    timestamp TIMESTAMPTZ DEFAULT NOW(),
    event_id VARCHAR(255),
    attempt_number INTEGER DEFAULT 1
);

-- Critical indexes for query performance
CREATE INDEX idx_smtp_failures_ip ON smtp_failures(sending_ip, timestamp);
CREATE INDEX idx_smtp_failures_domain ON smtp_failures(recipient_domain, timestamp);
CREATE INDEX idx_smtp_failures_timestamp ON smtp_failures(timestamp);
CREATE INDEX idx_smtp_failures_enhanced_code ON smtp_failures(enhanced_code);
```

**Architectural Decisions:**

- **No TTL**: Data accumulates forever
  - **Risk**: Table growth → slow queries → system degradation
  - **Recommended**: Add cleanup job to delete data > 30 days old
  - **3rd Order**: Without cleanup, indexes become bloated, slowing all queries

- **VARCHAR(10) for enhanced_code**: Sufficient for "5.7.1" format
  - **Risk**: If providers use longer codes, truncation occurs
  - **Mitigation**: Increase to VARCHAR(20) to be safe

### Table: ip_reputation_metrics

**Purpose**: Aggregated metrics per IP (current state)

```sql
CREATE TABLE ip_reputation_metrics (
    id SERIAL PRIMARY KEY,
    ip VARCHAR(45) UNIQUE NOT NULL,
    window_start TIMESTAMPTZ NOT NULL,
    window_end TIMESTAMPTZ NOT NULL,
    total_sent INTEGER DEFAULT 0,
    total_rejected INTEGER DEFAULT 0,
    rejection_ratio DECIMAL(5,4) DEFAULT 0.0000,
    unique_domains_rejected INTEGER DEFAULT 0,
    distinct_rejection_reasons JSONB DEFAULT '{}',
    major_providers_rejecting JSONB DEFAULT '[]',
    status VARCHAR(20) DEFAULT 'healthy',
    last_updated TIMESTAMPTZ DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'
);
```

**Architectural Decisions:**

- **UNIQUE constraint on IP**: Only one record per IP
  - **Pro**: Fast lookups, no duplicate states
  - **Con**: No historical tracking of metrics
  - **4th Order**: Cannot do trend analysis or detect gradual degradation

- **JSONB fields**: Flexible storage for error codes and provider lists
  - **Pro**: Can add new fields without migrations
  - **Con**: Cannot index into JSONB efficiently
  - **5th Order**: As data grows, queries filtering on JSONB become slow

### Table: ip_actions

**Purpose**: Audit log of all status changes and actions taken

```sql
CREATE TABLE ip_actions (
    id SERIAL PRIMARY KEY,
    ip VARCHAR(45) NOT NULL,
    action VARCHAR(50) NOT NULL,
    previous_status VARCHAR(20),
    new_status VARCHAR(20),
    reason TEXT,
    triggered_by VARCHAR(100) DEFAULT 'automated',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

**Critical for**:
- Audit compliance
- Incident investigation
- Understanding IP lifecycle

---

## Decision Algorithm

### Thresholds & Configuration

```go
type ReputationConfig struct {
    WindowMinutes                  int     // 15 minutes (rolling window)
    MinVolumeForAssessment         int     // 50 emails (prevents noise)
    BlacklistRejectionRatio        float64 // 0.05 (5%)
    BlacklistMinDomains            int     // 3 domains
    BlacklistMinMajorProviders     int     // 2 major providers
    QuarantineRejectionRatio       float64 // 0.03 (3%)
    QuarantineMinDomains           int     // 2 domains
    WarningRejectionRatio          float64 // 0.02 (2%)
    WarningReputationCodeThreshold int     // 5 codes
}
```

### Why These Thresholds?

**5% Blacklist Threshold**:
- Industry standard: Most email systems see 1-2% failure rate as normal
- 5% indicates clear systematic problem
- **3rd Order**: Too high risks reputation damage before detection
- **4th Order**: Too low causes alert fatigue and unnecessary IP swaps

**3 Domains Requirement**:
- Single domain rejection could be their issue
- Multiple domains indicates your problem
- **2nd Order**: Prevents false positives from single provider issues

**2 Major Providers**:
- Gmail, Outlook, Yahoo have sophisticated reputation systems
- If 2+ major providers reject, it's a real reputation problem
- **Critical 5th Order**: Major providers influence smaller providers' decisions

---

## Architectural Consequences

### 1. Race Conditions

**Scenario**: Aggregation runs while webhooks are being received

```
Thread 1: Webhook inserts failure at 10:15:01
Thread 2: Aggregation reads failures at 10:15:00
```

**Consequence**: Newly inserted failure missed by aggregation run

**Mitigation**: 
- PostgreSQL's MVCC prevents data corruption
- Failure will be caught in next aggregation (5 minutes max delay)
- **Acceptable**: 5-minute delay is within system tolerance

### 2. Missing Total Sent Metric

**CRITICAL ISSUE**: Currently estimates total_sent from failures

**Calculation**:
```go
estimated_sent = failure_count * 20  // Assumes 5% failure rate
```

**Problems**:
1. If actual failure rate is 10%, estimation is wrong by 2x
2. If actual failure rate is 1%, estimation is wrong by 5x
3. All thresholds become meaningless

**5th Order Consequences**:
- **Healthy IPs flagged as bad**: If estimation is low, ratios appear high
- **Bad IPs missed**: If estimation is high, ratios appear healthy
- **Lost trust**: Operators stop trusting system, ignore alerts
- **Business impact**: Either lose sending capacity or suffer deliverability issues

**REQUIRED FIX**: Integrate with Stalwart's sending metrics API

### 3. DNSBL Checks Performance

**Current**: 8 DNSBLs checked concurrently, 5-second timeout

**Consequences**:
- Best case: ~500ms (fastest DNSBL responds)
- Worst case: 5 seconds (timeout)
- Blocks goroutine during check

**Scaling Impact**:
- 100 IPs checked simultaneously = 100 goroutines
- If many timeouts occur: 500 goroutines waiting
- **4th Order**: Memory exhaustion → system crash

**Mitigation**: Add rate limiting and goroutine pool

### 4. No Event Deduplication

**Risk**: Stalwart retries webhook → duplicate events → inflated counts

**Detection**: Check for duplicate event_ids in database

**Fix**: Add UNIQUE constraint on event_id with ON CONFLICT DO NOTHING

---

## Testing & Visualization

See `/web/test-dashboard.html` for interactive testing interface.

All test scenarios are documented in `/scripts/test-ip-reputation.sh`.

---

**Next Steps for Production Readiness:**

1. ✅ Add comprehensive error code coverage (DONE)
2. ⚠️ Add event_id uniqueness constraint (CRITICAL)
3. ⚠️ Integrate with Stalwart sending metrics API (CRITICAL)
4. ⚠️ Add data retention policy (30-day cleanup)
5. ⚠️ Add rate limiting on webhook endpoint
6. ⚠️ Add DNSBL check rate limiting
7. ⚠️ Add historical metrics table for trend analysis
8. ⚠️ Add alerting integration (PagerDuty, Slack)


