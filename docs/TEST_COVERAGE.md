# Test Coverage Documentation

## Complete Test Suite: 15 Comprehensive Test Cases

This document maps all 15 SMTP error codes from the dashboard wiki to their respective test cases, ensuring complete coverage of every error scenario.

---

## 📋 Test Coverage Matrix

### PRIMARY Reputation Codes (Critical - Threshold: 2+)

| Error Code | Description | Test Case | Status Tested |
|------------|-------------|-----------|---------------|
| **5.7.1** | IP/domain reputation blocked | test-2, test-3, test-4, test-11, test-12 | warning, quarantine, blacklisted |
| **5.7.606** | Access denied (Microsoft-specific) | **test-11** ✨ NEW | quarantine |
| **5.7.512** | Message content rejected (spam) | **test-12** ✨ NEW | quarantine |

### AUTHENTICATION Codes (High - Threshold: 3+)

| Error Code | Description | Test Case | Status Tested |
|------------|-------------|-----------|---------------|
| **5.7.23** | SPF validation failed | test-7 | quarantine |
| **5.7.26** | Authentication required (ARC/DKIM) | **test-14** ✨ NEW | quarantine |

### INFRASTRUCTURE Codes (Medium - Threshold: 3+)

| Error Code | Description | Test Case | Status Tested |
|------------|-------------|-----------|---------------|
| **5.7.25** | PTR record required | test-8 | quarantine |
| **5.7.27** | Sender address has null MX | **test-13** ✨ NEW | quarantine |
| **5.7.7** | Domain has no MX/A/AAAA record | **test-13** ✨ NEW | quarantine |
| **5.1.8** | Bad sender's system address | **test-13** ✨ NEW | quarantine |

### POLICY & TEMPORARY Codes (Low - Threshold: 5+)

| Error Code | Description | Test Case | Status Tested |
|------------|-------------|-----------|---------------|
| **4.7.0** | Temporary rate limit/greylisting | test-3, test-6 | warning |
| **4.7.1** | Temporary policy rejection | **test-15** ✨ NEW | warning |
| **5.7.510** | Recipient address rejected (policy) | **test-15** ✨ NEW | warning |

### OTHER Common Codes (Informational)

| Error Code | Description | Test Case | Status Tested |
|------------|-------------|-----------|---------------|
| **5.1.1** | Recipient not found / Unknown user | test-1, test-2, test-5, test-9, test-10 | healthy, warning |
| **4.2.2** | Mailbox full | test-1, test-9 | healthy |
| **5.4.1** | Recipient address no longer available | **test-15** ✨ NEW | warning |

---

## 🎯 Test Cases Detailed Breakdown

### Test 1-10: Original Test Suite
These tests cover the fundamental scenarios and most common error codes.

### Test 11-15: NEW Extended Coverage ✨

#### **Test 11: Microsoft Reputation Block (5.7.606)**
- **Focus:** Microsoft-specific access denied scenarios
- **Error Codes:** 5.7.606 (primary), 5.7.1
- **Providers:** outlook.com, hotmail.com, live.com
- **Expected Status:** quarantine
- **Scenario:** When Microsoft specifically blocks your IP due to reputation issues
- **Real-world Impact:** Critical for email senders targeting Microsoft recipients

#### **Test 12: Content Spam Detection (5.7.512)**
- **Focus:** Message content flagged as spam
- **Error Codes:** 5.7.512 (primary), 5.7.1
- **Providers:** gmail.com, outlook.com, yahoo.com
- **Expected Status:** quarantine
- **Scenario:** Email content triggers spam filters
- **Real-world Impact:** Indicates compromised sending or poor content quality

#### **Test 13: Infrastructure DNS Issues (5.7.27, 5.7.7, 5.1.8)**
- **Focus:** Multiple DNS/MX configuration problems
- **Error Codes:** 5.7.27 (null MX), 5.7.7 (no MX), 5.1.8 (bad address)
- **Providers:** enterprise.com, business.net, corporate.org
- **Expected Status:** quarantine
- **Scenario:** DNS misconfiguration preventing email delivery
- **Real-world Impact:** Infrastructure issues requiring immediate DNS fixes

#### **Test 14: DKIM/ARC Authentication (5.7.26)**
- **Focus:** Advanced authentication failures (ARC, DKIM)
- **Error Codes:** 5.7.26
- **Providers:** gmail.com, yahoo.com
- **Expected Status:** quarantine
- **Scenario:** DKIM signatures missing or ARC validation failing
- **Real-world Impact:** Authentication issues affecting deliverability

#### **Test 15: Policy Rejections (4.7.1, 5.7.510, 5.4.1)**
- **Focus:** Temporary policy issues and recipient problems
- **Error Codes:** 4.7.1 (temp policy), 5.7.510 (policy reject), 5.4.1 (not available)
- **Providers:** gmail.com, outlook.com, yahoo.com
- **Expected Status:** warning
- **Scenario:** Mixed temporary issues that may resolve automatically
- **Real-world Impact:** Monitor for patterns but often self-correcting

---

## 🔄 Status Level Coverage

All 15 tests comprehensively cover the 4-tier status system:

| Status | Test Cases | Error Codes Tested |
|--------|------------|-------------------|
| **Healthy** | test-1, test-5, test-10 | 5.1.1, 4.2.2 (low volume) |
| **Warning** | test-2, test-6, test-9, **test-15** | 5.7.1, 4.7.0, 4.7.1, 5.7.510, 5.4.1 |
| **Quarantine** | test-3, test-7, test-8, **test-11, test-12, test-13, test-14** | All critical codes |
| **Blacklisted** | test-4 | 5.7.1 (high volume, multiple providers) |

---

## 📊 Error Code Category Coverage

### ✅ Complete Coverage Achieved

- **PRIMARY Codes:** 3/3 codes covered (100%)
  - 5.7.1, 5.7.606, 5.7.512

- **AUTHENTICATION Codes:** 2/2 codes covered (100%)
  - 5.7.23, 5.7.26

- **INFRASTRUCTURE Codes:** 4/4 codes covered (100%)
  - 5.7.25, 5.7.27, 5.7.7, 5.1.8

- **POLICY Codes:** 3/3 codes covered (100%)
  - 4.7.0, 4.7.1, 5.7.510

- **OTHER Codes:** 3/3 codes covered (100%)
  - 5.1.1, 4.2.2, 5.4.1

**Total: 15/15 error codes covered (100% coverage)** ✅

---

## 🧪 Running the Tests

### Web Dashboard (Recommended)
```bash
open web/test-dashboard.html
```
Click "▶️ Run All Tests" to execute all 15 test cases with beautiful visual results.

### Command Line
```bash
./scripts/test-ip-reputation.sh
```

### API
```bash
# Run all tests
curl -X POST http://localhost:8080/api/testing/test-suite/run | jq '.'

# View all test cases
curl http://localhost:8080/api/testing/test-cases | jq '.'

# Run specific test
curl -X POST http://localhost:8080/api/testing/test-cases/test-11/run | jq '.'
```

---

## 🔍 Validation Checklist

When running the test suite, you should see:

- ✅ **15 test cases** loaded
- ✅ **All 15 error codes** represented
- ✅ **4 status levels** tested (healthy, warning, quarantine, blacklisted)
- ✅ **Multiple major providers** tested (Gmail, Outlook, Yahoo, etc.)
- ✅ **Edge cases** covered (low volume, mixed signals, gradual decay)
- ✅ **Real-world scenarios** simulated

---

## 🎯 Key Insights

### Why 15 Tests?

1. **Complete Error Code Coverage:** Every error code from the wiki is tested
2. **Multiple Scenarios:** Same codes tested in different contexts
3. **Real-world Patterns:** Tests mirror actual delivery failure patterns
4. **Decision Algorithm Validation:** Ensures all thresholds work correctly
5. **Provider Diversity:** Tests across different email providers

### Critical vs Non-Critical Codes

**Immediate Action Required (PRIMARY):**
- 5.7.1, 5.7.606, 5.7.512 → Threshold: 2+ occurrences

**Investigation Needed (AUTH/INFRA):**
- 5.7.23, 5.7.26, 5.7.25, 5.7.27, 5.7.7, 5.1.8 → Threshold: 3+ occurrences

**Monitor Closely (POLICY):**
- 4.7.0, 4.7.1, 5.7.510 → Threshold: 5+ occurrences (often temporary)

---

## 📈 Expected Results

All 15 tests should pass with 100% success rate:

```json
{
  "total_tests": 15,
  "passed_tests": 15,
  "failed_tests": 0,
  "execution_time_ms": 150-300
}
```

If any test fails:
1. Check database connectivity
2. Verify aggregation service is running
3. Review logs for error details
4. Check decision algorithm configuration

---

## 🔗 Related Documentation

- **[Dashboard Wiki](../web/test-dashboard.html)** - Complete error code reference
- **[Decision Algorithm](../internal/reputation/decision.go)** - Status determination logic
- **[Test Handler](../internal/api/test_suite_handlers.go)** - Test implementation
- **[TECHNICAL_REFERENCE.md](TECHNICAL_REFERENCE.md)** - System architecture
- **[DEVELOPER_GUIDE.md](../guides/DEVELOPER_GUIDE.md)** - Development guide

---

**Last Updated:** 2025-11-16  
**Test Suite Version:** 2.0  
**Coverage:** 15/15 error codes (100%)

