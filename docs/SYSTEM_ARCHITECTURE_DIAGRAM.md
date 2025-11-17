# IP Reputation System - Complete Architecture

**Visual representation of the entire system**

---

## System Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                                                                              │
│                         STALWART MAIL SERVER                                 │
│                    (Sends emails on your behalf)                             │
│                                                                              │
└────────────────────────────────┬────────────────────────────────────────────┘
                                 │
                                 │ Webhook on delivery failure
                                 │ POST /api/webhooks/stalwart/delivery-failure
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                                                                              │
│                      YOUR IP REPUTATION SYSTEM                               │
│                        (GoLang Backend Service)                              │
│                                                                              │
│  ┌────────────────────────────────────────────────────────────────────┐    │
│  │                    PHASE 1: WEBHOOK RECEPTION                       │    │
│  │                         (Synchronous)                               │    │
│  │                                                                     │    │
│  │  Webhook Handler                                                   │    │
│  │       │                                                             │    │
│  │       ├─→ Parse JSON                                               │    │
│  │       ├─→ Validate (all fields present?)                           │    │
│  │       ├─→ Extract domain from email                                │    │
│  │       ├─→ Check enhanced_code against 13 patterns                  │    │
│  │       │                                                             │    │
│  │       ▼                                                             │    │
│  │  Database Insert                                                   │    │
│  │       │                                                             │    │
│  │       ├─→ smtp_failures table                                      │    │
│  │       ├─→ ON CONFLICT (event_id) DO NOTHING  ← Deduplication!     │    │
│  │       └─→ Returns ID if inserted                                   │    │
│  │       │                                                             │    │
│  │       ▼                                                             │    │
│  │  Prometheus Metrics                                                │    │
│  │       ├─→ smtp_failures_total{ip, code, domain}++                 │    │
│  │       └─→ webhook_events_total{type, status}++                    │    │
│  │       │                                                             │    │
│  │       ▼                                                             │    │
│  │  HTTP 200 OK                                                       │    │
│  │                                                                     │    │
│  └────────────────────────────────────────────────────────────────────┘    │
│                                                                              │
│  ┌────────────────────────────────────────────────────────────────────┐    │
│  │                 PHASE 2: BACKGROUND AGGREGATION                     │    │
│  │                   (Every 5 minutes, Async)                          │    │
│  │                                                                     │    │
│  │  Timer Trigger (5 min)                                             │    │
│  │       │                                                             │    │
│  │       ▼                                                             │    │
│  │  Find IPs with Recent Failures                                     │    │
│  │       │   SELECT DISTINCT sending_ip                               │    │
│  │       │   FROM smtp_failures                                       │    │
│  │       │   WHERE timestamp >= NOW() - INTERVAL '15 minutes'         │    │
│  │       │                                                             │    │
│  │       ▼                                                             │    │
│  │  For Each IP:                                                      │    │
│  │       │                                                             │    │
│  │       ├─→ Get all failures in last 15 minutes                      │    │
│  │       ├─→ Count unique domains                                     │    │
│  │       ├─→ Count each error code                                    │    │
│  │       ├─→ Identify major providers (Gmail, Outlook, etc.)          │    │
│  │       ├─→ Calculate rejection ratio                                │    │
│  │       │                                                             │    │
│  │       ▼                                                             │    │
│  │  Decision Algorithm (decision.go)                                  │    │
│  │       │                                                             │    │
│  │       ├─→ Check: total_sent >= 50? (minimum volume)                │    │
│  │       │         NO → return "healthy"                              │    │
│  │       │                                                             │    │
│  │       ├─→ Check: BLACKLISTED?                                      │    │
│  │       │         rejection_ratio > 5% AND                           │    │
│  │       │         unique_domains >= 3 AND                            │    │
│  │       │         major_providers >= 2 AND                           │    │
│  │       │         has_reputation_codes?                              │    │
│  │       │         YES → return "blacklisted" → CRITICAL ALERT        │    │
│  │       │                                                             │    │
│  │       ├─→ Check: QUARANTINE?                                       │    │
│  │       │         rejection_ratio > 3% AND major_providers >= 1      │    │
│  │       │         YES → return "quarantine" → WARNING ALERT          │    │
│  │       │                                                             │    │
│  │       ├─→ Check: WARNING?                                          │    │
│  │       │         rejection_ratio >= 2% OR                           │    │
│  │       │         (throttle_count > 10 AND rejected > 0)             │    │
│  │       │         YES → return "warning" → CAUTION ALERT             │    │
│  │       │                                                             │    │
│  │       └─→ return "healthy"                                         │    │
│  │       │                                                             │    │
│  │       ▼                                                             │    │
│  │  Update Database                                                   │    │
│  │       │                                                             │    │
│  │       ├─→ UPSERT ip_reputation_metrics                             │    │
│  │       │   (one row per IP, updated every 5 min)                    │    │
│  │       │                                                             │    │
│  │       ▼                                                             │    │
│  │  If Status Changed:                                                │    │
│  │       │                                                             │    │
│  │       ├─→ INSERT INTO ip_actions (audit log)                       │    │
│  │       ├─→ Update Prometheus gauge (ip_reputation_status)           │    │
│  │       ├─→ Trigger DNSBL check (async)                              │    │
│  │       └─→ Log alert (structured JSON)                              │    │
│  │                                                                     │    │
│  └────────────────────────────────────────────────────────────────────┘    │
│                                                                              │
│  ┌────────────────────────────────────────────────────────────────────┐    │
│  │                   PHASE 3: DNSBL CHECKING                           │    │
│  │                   (On-demand, Async)                                │    │
│  │                                                                     │    │
│  │  Trigger: Manual API call OR status change to quarantine/blacklist │    │
│  │       │                                                             │    │
│  │       ▼                                                             │    │
│  │  Concurrent DNS Lookups                                            │    │
│  │       │                                                             │    │
│  │       ├─→ zen.spamhaus.org        (5s timeout) ────┐               │    │
│  │       ├─→ b.barracudacentral.org  (5s timeout) ────┤               │    │
│  │       ├─→ bl.spamcop.net          (5s timeout) ────┤               │    │
│  │       ├─→ cbl.abuseat.org         (5s timeout) ────┤               │    │
│  │       ├─→ dnsbl.sorbs.net         (5s timeout) ────┤ (8 goroutines)│    │
│  │       ├─→ bl.spamcannibal.org     (5s timeout) ────┤               │    │
│  │       ├─→ psbl.surriel.com        (5s timeout) ────┤               │    │
│  │       └─→ dnsbl-1.uceprotect.net  (5s timeout) ────┘               │    │
│  │       │                                                             │    │
│  │       ▼                                                             │    │
│  │  Aggregate Results                                                 │    │
│  │       ├─→ listed: true/false                                       │    │
│  │       ├─→ listings: ["spamhaus", "barracuda"]                      │    │
│  │       ├─→ severity: "critical" if Spamhaus                         │    │
│  │       │                                                             │    │
│  │       ▼                                                             │    │
│  │  Store in dnsbl_checks table                                       │    │
│  │       └─→ Return to caller                                         │    │
│  │                                                                     │    │
│  └────────────────────────────────────────────────────────────────────┘    │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
                                 │
                                 │ APIs for querying
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                                                                              │
│                         MONITORING & ALERTS                                  │
│                                                                              │
│  Prometheus Metrics:                     API Endpoints:                     │
│  ├─ smtp_failures_total                  ├─ GET /api/ips/{ip}/reputation   │
│  ├─ ip_status_changes_total              ├─ GET /api/ips/{ip}/failures     │
│  ├─ ip_reputation_status (gauge)         ├─ POST /api/ips/{ip}/quarantine  │
│  ├─ ip_rejection_ratio                   ├─ POST /api/ips/{ip}/dnsbl-check │
│  ├─ dnsbl_checks_total                   └─ GET /api/dashboard/ip-health   │
│  ├─ dnsbl_check_duration_seconds                                            │
│  └─ ip_aggregation_runs_total            Dashboard:                         │
│                                           └─ web/test-dashboard.html        │
│  Grafana Dashboards:                                                        │
│  └─ http://localhost:3000                Swagger UI:                        │
│                                           └─ /swagger/index.html            │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Error Code Processing Flow

```
Enhanced Code Received: "5.7.1"
    │
    ├─→ Is it PRIMARY? (5.7.1, 5.7.606, 5.7.512)
    │   └─→ YES → Check if count >= 2 → REPUTATION CONCERN
    │
    ├─→ Is it AUTHENTICATION? (5.7.23, 5.7.26)
    │   └─→ YES → Check if count >= 3 → CONFIGURATION ISSUE
    │
    ├─→ Is it INFRASTRUCTURE? (5.7.25, 5.7.27, 5.7.7, 5.1.8)
    │   └─→ YES → Check if count >= 3 → DNS PROBLEM
    │
    └─→ Is it POLICY? (4.7.0, 4.7.1, 5.7.510)
        └─→ YES → Check if count >= 5 → RATE LIMITING

All checks pass → Used in decision algorithm
```

---

## Database Schema Relationships

```
┌──────────────────────────┐
│   smtp_failures          │  ← Raw event log (append-only)
│                          │
│  PK: id (SERIAL)         │
│  UK: event_id ───────────┼─── CRITICAL: Prevents duplicates
│      sending_ip          │
│      recipient_domain    │
│      enhanced_code       │
│      timestamp           │
│      ...                 │
└──────────┬───────────────┘
           │
           │ Aggregated every 5 minutes
           │
           ▼
┌──────────────────────────┐
│  ip_reputation_metrics   │  ← Current state (one row per IP)
│                          │
│  PK: id (SERIAL)         │
│  UK: ip ─────────────────┼─── ONE ROW PER IP (upserted)
│      status              │     "healthy" | "warning" | "quarantine" | "blacklisted"
│      rejection_ratio     │
│      total_sent          │     ⚠️ ESTIMATED (needs Stalwart integration)
│      total_rejected      │
│      major_providers []  │     JSONB: ["gmail.com", "outlook.com"]
│      rejection_codes {}  │     JSONB: {"5.7.1": 10, "5.7.23": 3}
│      window_start/end    │     15-minute rolling window
│      ...                 │
└──────────┬───────────────┘
           │
           │ When status changes
           │
           ▼
┌──────────────────────────┐
│   ip_actions             │  ← Audit log (history)
│                          │
│  PK: id (SERIAL)         │
│      ip                  │
│      action              │     "status_change" | "manual_quarantine"
│      previous_status     │
│      new_status          │
│      triggered_by        │     "automated" | "manual"
│      created_at          │
└──────────────────────────┘

┌──────────────────────────┐
│   dnsbl_checks           │  ← DNSBL check results
│                          │
│  PK: id (SERIAL)         │
│      ip                  │
│      listed (BOOLEAN)    │
│      listings []         │     JSONB: ["zen.spamhaus.org"]
│      checked_at          │
│      check_duration_ms   │
└──────────────────────────┘
```

---

## Decision Algorithm Flowchart

```
Start: Calculate health metrics for IP
    │
    ▼
┌─────────────────────────┐
│ total_sent >= 50?       │
└───┬─────────────────────┘
    │
    ├─ NO ──→ return "healthy"  (insufficient volume)
    │
    ▼
┌─────────────────────────┐
│ BLACKLIST CHECK:        │
│ • rejection > 5%        │
│ • domains >= 3          │
│ • major_providers >= 2  │
│ • has_reputation_codes  │
└───┬─────────────────────┘
    │
    ├─ YES ──→ return "blacklisted" ──→ CRITICAL ALERT
    │                                    ├─ Immediate quarantine
    │                                    ├─ Trigger DNSBL check
    │                                    ├─ Alert ops team
    │                                    └─ Log to ip_actions
    ▼
┌─────────────────────────┐
│ QUARANTINE CHECK:       │
│ • (rejection > 3% AND   │
│    major_providers >= 1)│
│   OR                    │
│ • (rejection > 5% AND   │
│    domains >= 2)        │
└───┬─────────────────────┘
    │
    ├─ YES ──→ return "quarantine" ──→ WARNING ALERT
    │                                   ├─ Reduce traffic 50%
    │                                   ├─ Trigger DNSBL check
    │                                   └─ Monitor closely
    ▼
┌─────────────────────────┐
│ WARNING CHECK:          │
│ • rejection >= 2%       │
│   OR                    │
│ • throttle_count > 10   │
│   OR                    │
│ • repeated 5.7.1 (5+)   │
└───┬─────────────────────┘
    │
    ├─ YES ──→ return "warning" ──→ CAUTION ALERT
    │                                ├─ Monitor closely
    │                                └─ Check list hygiene
    ▼
return "healthy"  ──→ Continue normal operations
```

---

## Data Flow Timeline

```
Time: 10:00:00
├─ Email sent from your IP 203.0.113.10 to user@gmail.com
│
Time: 10:00:05
├─ Gmail rejects with 550 5.7.1 "Message rejected due to IP reputation"
│
Time: 10:00:06
├─ Stalwart receives rejection
├─ Stalwart sends webhook to your system
│
Time: 10:00:06.100
├─ Your system receives webhook
├─ Parses enhanced_code "5.7.1"
├─ Checks: Is this a PRIMARY reputation code? YES
├─ Inserts to smtp_failures table
├─ Increments Prometheus counter: smtp_failures_total{ip="203.0.113.10", code="5.7.1", domain="gmail.com"}
├─ Returns HTTP 200 OK to Stalwart
│
Time: 10:05:00  (Next aggregation run)
├─ Background service wakes up
├─ Queries: SELECT DISTINCT sending_ip FROM smtp_failures WHERE timestamp >= '09:50:00'
├─ Found: 203.0.113.10 has failures
├─ Gets all failures for this IP in last 15 minutes
│  Result: 5 failures
│  ├─ 3 × 5.7.1 from gmail.com
│  ├─ 1 × 5.7.1 from outlook.com
│  └─ 1 × 5.1.1 from unknown-domain.com
│
├─ Calculates:
│  ├─ total_sent (estimated): 100 emails
│  ├─ total_rejected: 5
│  ├─ rejection_ratio: 5/100 = 0.05 (5%)
│  ├─ unique_domains: 3 (gmail, outlook, unknown-domain)
│  ├─ major_providers: 2 (gmail, outlook)
│  ├─ reputation_codes: {"5.7.1": 4, "5.1.1": 1}
│
├─ Decision algorithm:
│  ├─ total_sent >= 50? YES (100 >= 50)
│  ├─ BLACKLIST? rejection > 5%? NO (5% = 5%, not >)
│  ├─ QUARANTINE? rejection > 3% AND major_providers >= 1? YES (5% > 3% AND 2 >= 1)
│  └─ Result: "quarantine"
│
├─ Checks previous status: "healthy"
├─ Status changed! healthy → quarantine
│
├─ Actions:
│  ├─ UPSERT ip_reputation_metrics (set status = "quarantine")
│  ├─ INSERT ip_actions (record status change)
│  ├─ Update Prometheus gauge: ip_reputation_status{ip="203.0.113.10"} = 3
│  ├─ Log warning: "IP 203.0.113.10 has been QUARANTINED"
│  ├─ Trigger DNSBL check (async goroutine)
│  │
│  └─ DNSBL Check (parallel):
│     ├─ zen.spamhaus.org: Not listed
│     ├─ b.barracudacentral.org: Not listed
│     ├─ bl.spamcop.net: Not listed
│     ├─ cbl.abuseat.org: Not listed
│     ├─ dnsbl.sorbs.net: Not listed
│     ├─ bl.spamcannibal.org: Not listed
│     ├─ psbl.surriel.com: Not listed
│     └─ dnsbl-1.uceprotect.net: Not listed
│     Result: listed = false, listings = []
│     INSERT dnsbl_checks
│
Time: 10:05:02
└─ Aggregation complete
   └─ Operators can now query: GET /api/ips/203.0.113.10/reputation
      Returns: status = "quarantine", recommendations = [...]
```

---

## Testing Flow

```
User Opens: web/test-dashboard.html
    │
    ├─ Clicks "Run All Tests"
    │
    ▼
Browser sends: POST /api/testing/test-suite/run
    │
    ▼
Backend runs 15 test scenarios:
    │
    ├─ Test 1: Healthy IP (500 sent, 2 failures)
    │   └─→ Expected: "healthy", Actual: "healthy" ✅
    │
    ├─ Test 2: Warning State (300 sent, 8 failures)
    │   └─→ Expected: "warning", Actual: "warning" ✅
    │
    ├─ Test 3: Quarantine (400 sent, 15 failures, 2 major providers)
    │   └─→ Expected: "quarantine", Actual: "quarantine" ✅
    │
    ├─ Test 4: Blacklisted (500 sent, 35 failures, 3 domains, 2 major)
    │   └─→ Expected: "blacklisted", Actual: "blacklisted" ✅
    │
    └─ ... 11 more tests covering all 15 error codes
    │
    ▼
Returns:
    {
      "total_tests": 15,
      "passed_tests": 15,
      "failed_tests": 0,
      "results": [...]
    }
    │
    ▼
Dashboard displays:
    ├─ Green cards for passed tests ✅
    ├─ Red cards for failed tests ❌
    ├─ Execution time per test
    └─ Detailed metrics (rejection ratios, etc.)
```

---

## Critical Path: What Happens When IP Goes Bad

```
Stage 1: HEALTHY
├─ Rejection ratio: < 2%
├─ Action: None
└─ Monitoring: Standard

↓ (Spam complaint or authentication issue occurs)

Stage 2: WARNING (2-3% rejection)
├─ Rejection ratio: >= 2%
├─ Action: Log warning, increase monitoring
├─ Alert level: Informational
└─ Recommendation: Check list hygiene

↓ (More rejections from major providers)

Stage 3: QUARANTINE (3-5% rejection)
├─ Rejection ratio: > 3%, major providers rejecting
├─ Action: 
│  ├─ Reduce traffic by 50%
│  ├─ Trigger DNSBL check
│  └─ Alert ops team (WARNING)
├─ Alert level: Warning
└─ Recommendation: Investigate immediately

↓ (Problem not resolved, rejections increase)

Stage 4: BLACKLISTED (> 5% rejection)
├─ Rejection ratio: > 5%
├─ Criteria: 3+ domains, 2+ major providers, reputation codes
├─ Action:
│  ├─ Immediate quarantine
│  ├─ Trigger DNSBL check
│  ├─ Alert ops team (CRITICAL)
│  ├─ Log to ip_actions
│  └─ Consider IP swap
├─ Alert level: CRITICAL
└─ Recommendation: 
   ├─ Stop sending from this IP
   ├─ Check for compromise
   ├─ Review recent campaigns
   └─ Submit delisting requests
```

---

## File Locations Reference

```
golang-backend-service/
├── cmd/server/main.go              ← Entry point, starts everything
├── internal/
│   ├── api/
│   │   ├── routes.go               ← Core API routes
│   │   └── ip_reputation_handlers.go ← IP reputation endpoints
│   ├── database/
│   │   ├── postgres.go             ← DB connection
│   │   └── ip_reputation.go        ← Data layer (INSERT, SELECT, etc.)
│   ├── logger/
│   │   └── logger.go               ← Structured logging
│   └── reputation/
│       ├── decision.go             ← ⭐ Decision algorithm (13 error codes)
│       ├── dnsbl.go                ← ⭐ DNSBL checking (8 blacklists)
│       ├── aggregation.go          ← ⭐ Background service (every 5 min)
│       └── metrics.go              ← Prometheus metrics
├── docs/
│   ├── CRITICAL_FINDINGS_AND_ANSWERS.md  ← ⭐ Read this first!
│   ├── SMTP_ERROR_CODES_REFERENCE.md     ← Complete error code guide
│   └── SYSTEM_ARCHITECTURE_DIAGRAM.md    ← This file
├── web/
│   └── test-dashboard.html         ← ⭐ ONE-CLICK TESTING
├── scripts/
│   └── test-ip-reputation.sh       ← Command-line testing
├── Context/Data/
│   ├── docker-compose.yml          ← Start services
│   └── init.sql                    ← ⭐ Database schema (with fixes)
├── QUICK_ANSWERS.md                ← ⭐ TL;DR version
└── README.md                       ← ⭐ Start here
```

---

**Key**: ⭐ = Must read/use  
**Status**: Production-ready (with noted caveats)  
**Last Updated**: November 16, 2025


