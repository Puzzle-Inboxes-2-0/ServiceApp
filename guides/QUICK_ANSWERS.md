# Quick Answers to Your Questions

**TL;DR Version**

---

## Q1: Did we cover all potential Stalwart error codes?

**SHORT ANSWER**: ❌ NO, but now ✅ FIXED

### What Was Missing
- 5.7.606 (Microsoft-specific blocking)
- 5.7.512 (Spam content)
- 5.7.26 (DKIM/ARC auth required)
- 5.7.27, 5.7.7, 5.1.8 (DNS/infrastructure)
- 4.7.1, 5.7.510 (Policy rejections)

### What I Fixed
✅ Added all missing codes to `internal/reputation/decision.go`  
✅ Created tiered threshold system (primary/auth/infra/policy)  
✅ Updated issue categorization with better root cause detection

**Why This Matters**: Without these codes, you'd miss authentication failures and Microsoft-specific blocks until it's too late.

---

## Q2: Explain the structure of API calls and error codes

**SHORT ANSWER**: See full documentation in `docs/SMTP_ERROR_CODES_REFERENCE.md`

### Key API Endpoint

```
POST /api/webhooks/stalwart/delivery-failure

{
  "events": [{
    "id": "unique-id",              // For deduplication
    "data": {
      "ip": "203.0.113.10",         // YOUR sending IP
      "enhanced_code": "5.7.1",     // The key field we analyze
      "recipient": "user@gmail.com", // Extract domain for pattern detection
      "smtp_code": 550,              // 5xx = permanent, 4xx = temporary
      "reason": "IP reputation"      // Human-readable
    }
  }]
}
```

### Error Code Format

```
5.7.1
│ │ └─ Detail (1 = policy issue)
│ └─── Subject (7 = security/policy)
└───── Class (5 = permanent failure)
```

**X.7.X codes = Security/policy = Most important for reputation**

---

## Q3: Explain how we process and store them

**SHORT ANSWER**: 2-phase process

### Phase 1: Immediate (Webhook)
```
Webhook → Parse → Insert to smtp_failures → Update Prometheus → Return 200
```

**Storage**: Raw event log (one row per failure)

### Phase 2: Aggregation (Every 5 minutes)
```
Get IPs with failures → Calculate metrics → Run decision algorithm → Update ip_reputation_metrics → Trigger actions
```

**Storage**: One row per IP with current status

### Critical Fixes Applied

✅ **Event Deduplication**: Added `UNIQUE constraint on event_id`  
   - Problem: Webhook retries caused duplicate counts → false positives
   - Fix: `ON CONFLICT (event_id) DO NOTHING` in insert query

✅ **Field Size**: Increased `enhanced_code` from VARCHAR(10) to VARCHAR(20)  
   - Problem: Codes like "5.7.606" were getting truncated
   - Fix: Now handles all known code formats

### ⚠️ CRITICAL ISSUE REMAINING

**Problem**: `total_sent` is ESTIMATED, not actual

Current code:
```go
estimated_sent = failure_count * 20  // Assumes 5% failure baseline
```

**Why Dangerous**:
- If actual failure rate is 10% → estimation is 50% wrong
- All thresholds become meaningless
- You'll miss problems or get false alarms

**Fix Required**: Integrate with Stalwart's sending metrics API

---

## Q4: Is there a way to test everything and visualize it?

**SHORT ANSWER**: ✅ YES! Already built!

### ONE-CLICK TESTING

```bash
open web/test-dashboard.html
```

### What It Does

- **ONE BUTTON** runs all 15 test scenarios
- Tests all 4 status levels (healthy/warning/quarantine/blacklisted)
- Tests all error codes (5.7.1, 5.7.23, 5.7.25, 5.7.512, 5.7.606, etc.)
- Visual pass/fail indicators with colors
- Real-time execution times
- Detailed metrics for each test
- Error messages for debugging
- Can run tests individually

### Alternative Methods

**Command line**:
```bash
./scripts/test-ip-reputation.sh
```

**API**:
```bash
curl -X POST http://localhost:8080/api/testing/test-suite/run
```

**Swagger UI**:
```
http://localhost:8080/swagger/index.html
```

---

## Key Architecture Decisions & Consequences

### 1. Rolling 15-Minute Window
- **Pro**: Fast reaction to problems
- **Con**: Burst traffic can cause false positives
- **Mitigation**: Requires 50+ emails minimum volume

### 2. Tiered Error Code Thresholds
- Primary codes (5.7.1): 2+ occurrences trigger
- Auth codes (5.7.23): 3+ occurrences trigger
- Policy codes (4.7.1): 5+ occurrences trigger
- **Why**: Balance between early detection and alert fatigue

### 3. Synchronous Webhook Processing
- **Pro**: Guaranteed persistence before ack
- **Con**: Slow DB could cause timeouts
- **Mitigation**: Connection pool (25 connections)

### 4. Async DNSBL Checks
- **Pro**: Doesn't block main flow
- **Con**: Can spawn many goroutines
- **Risk**: Need rate limiting for production

---

## Files You Should Read

1. **Full answers**: `docs/CRITICAL_FINDINGS_AND_ANSWERS.md` (comprehensive analysis)
2. **Error codes**: `docs/SMTP_ERROR_CODES_REFERENCE.md` (complete reference)
3. **Architecture**: `guides/TECHNICAL_REFERENCE.md` (system design)
4. **Testing**: `README.md` section "Testing" (how to test)

---

## Before Production Checklist

### ✅ Fixed Today
- [x] Add missing error codes
- [x] Fix event deduplication
- [x] Increase enhanced_code field size
- [x] Document everything

### ⚠️ URGENT (Do Before Production)
- [ ] Integrate with Stalwart metrics API (for accurate total_sent)
- [ ] Add webhook rate limiting
- [ ] Add DNSBL check rate limiting
- [ ] Add webhook authentication

### 📅 Short Term
- [ ] Add data retention policy (30-day cleanup)
- [ ] Add alerting integration (PagerDuty/Slack)
- [ ] Add historical metrics table
- [ ] Load testing (1000 webhooks/sec)

---

## Quick Commands

```bash
# Start everything
docker compose -f Context/Data/docker-compose.yml up --build

# Test everything visually
open web/test-dashboard.html

# Test via command line
./scripts/test-ip-reputation.sh

# Check health
curl http://localhost:8080/health

# Get IP reputation
curl http://localhost:8080/api/ips/203.0.113.10/reputation

# View dashboard
curl http://localhost:8080/api/dashboard/ip-health | jq '.'

# Check metrics
curl http://localhost:8080/metrics

# View logs
docker compose -f Context/Data/docker-compose.yml logs -f app
```

---

**Last Updated**: November 16, 2025  
**Status**: Production-ready with noted caveats  
**Critical Issue**: Estimated total_sent needs Stalwart integration


