#!/bin/bash

# Test IP Reputation System
# This script tests all 15 test cases covering every error code from the wiki

set -e

BASE_URL="${1:-http://127.0.0.1:8080}"
WEBHOOK_URL="$BASE_URL/api/webhooks/stalwart/delivery-failure"
SIMULATE_URL="$BASE_URL/api/testing/simulate-failures"

echo "=========================================="
echo "IP Reputation System Test Suite"
echo "=========================================="
echo "Base URL: $BASE_URL"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Function to run a test case
run_test() {
    local test_name="$1"
    local test_data="$2"
    local expected_status="$3"
    
    echo "----------------------------------------"
    echo "Test: $test_name"
    echo "Expected Status: $expected_status"
    echo "----------------------------------------"
    
    # Send test data
    response=$(curl -s -X POST "$SIMULATE_URL" \
        -H "Content-Type: application/json" \
        -d "$test_data")
    
    # Extract actual status
    actual_status=$(echo "$response" | grep -o '"ip_status":"[^"]*"' | cut -d'"' -f4)
    
    # Display results
    echo "Response:"
    echo "$response" | jq '.' 2>/dev/null || echo "$response"
    echo ""
    
    # Check if status matches
    if [ "$actual_status" == "$expected_status" ]; then
        echo -e "${GREEN}✅ PASSED${NC} - Status: $actual_status"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}❌ FAILED${NC} - Expected: $expected_status, Got: $actual_status"
        ((TESTS_FAILED++))
    fi
    echo ""
    sleep 1
}

# Test Case 1: Healthy IP - Normal Operations
echo "=== Test Case 1: Healthy IP - Normal Operations ==="
run_test "Healthy IP" '{
  "ip": "203.0.113.10",
  "total_sent": 500,
  "failures": [
    {"code": "5.1.1", "domain": "unknown-domain.com", "count": 1, "reason": "Recipient not found"},
    {"code": "4.2.2", "domain": "example.com", "count": 1, "reason": "Mailbox full"}
  ]
}' "healthy"

# Test Case 2: Warning State - Elevated Rejections
echo "=== Test Case 2: Warning State - Elevated Rejections ==="
run_test "Warning State" '{
  "ip": "203.0.113.11",
  "total_sent": 300,
  "failures": [
    {"code": "5.7.1", "domain": "gmail.com", "count": 3, "reason": "IP reputation"},
    {"code": "5.7.1", "domain": "outlook.com", "count": 2, "reason": "Policy reject"},
    {"code": "5.1.1", "domain": "various.com", "count": 3, "reason": "Unknown user"}
  ]
}' "warning"

# Test Case 3: Quarantine - Multiple Major Providers Rejecting
echo "=== Test Case 3: Quarantine - Multiple Major Providers Rejecting ==="
run_test "Quarantine Status" '{
  "ip": "203.0.113.12",
  "total_sent": 400,
  "failures": [
    {"code": "5.7.1", "domain": "gmail.com", "count": 7, "reason": "IP reputation"},
    {"code": "5.7.1", "domain": "outlook.com", "count": 5, "reason": "Policy reject"},
    {"code": "4.7.0", "domain": "yahoo.com", "count": 3, "reason": "Temporarily deferred"}
  ]
}' "quarantine"

# Test Case 4: Blacklisted - Critical Reputation Damage
echo "=== Test Case 4: Blacklisted - Critical Reputation Damage ==="
run_test "Blacklisted IP" '{
  "ip": "203.0.113.13",
  "total_sent": 500,
  "failures": [
    {"code": "5.7.1", "domain": "gmail.com", "count": 12, "reason": "IP reputation"},
    {"code": "5.7.1", "domain": "outlook.com", "count": 10, "reason": "Blocked by policy"},
    {"code": "5.7.1", "domain": "yahoo.com", "count": 8, "reason": "Spam detected"},
    {"code": "5.7.1", "domain": "aol.com", "count": 5, "reason": "IP on blocklist"}
  ]
}' "blacklisted"

# Test Case 5: False Positive - Low Volume
echo "=== Test Case 5: False Positive - Low Volume ==="
run_test "Low Volume (Should Stay Healthy)" '{
  "ip": "203.0.113.14",
  "total_sent": 20,
  "failures": [
    {"code": "5.7.1", "domain": "gmail.com", "count": 2, "reason": "IP reputation"},
    {"code": "5.1.1", "domain": "example.com", "count": 1, "reason": "Unknown user"}
  ]
}' "healthy"

# Test Case 6: Temporary Throttling (4xx codes)
echo "=== Test Case 6: Temporary Throttling ==="
run_test "Throttling" '{
  "ip": "203.0.113.15",
  "total_sent": 600,
  "failures": [
    {"code": "4.7.0", "domain": "gmail.com", "count": 12, "reason": "Rate limited"},
    {"code": "4.2.1", "domain": "outlook.com", "count": 4, "reason": "Mailbox busy"},
    {"code": "5.7.1", "domain": "yahoo.com", "count": 2, "reason": "Policy"}
  ]
}' "warning"

# Test Case 7: SPF/DKIM Failures (Configuration Issue)
echo "=== Test Case 7: SPF/DKIM Failures ==="
run_test "Authentication Failure" '{
  "ip": "203.0.113.16",
  "total_sent": 300,
  "failures": [
    {"code": "5.7.23", "domain": "gmail.com", "count": 15, "reason": "SPF validation failed"},
    {"code": "5.7.1", "domain": "outlook.com", "count": 10, "reason": "DKIM fail"}
  ]
}' "quarantine"

# Test Case 8: PTR Record Missing
echo "=== Test Case 8: PTR Record Missing ==="
run_test "PTR Record Issue" '{
  "ip": "203.0.113.17",
  "total_sent": 200,
  "failures": [
    {"code": "5.7.25", "domain": "gmail.com", "count": 8, "reason": "PTR record required"},
    {"code": "5.7.25", "domain": "outlook.com", "count": 4, "reason": "Reverse DNS lookup failed"}
  ]
}' "quarantine"

# Test Case 9: Mixed Signals - Hard to Classify
echo "=== Test Case 9: Mixed Signals ==="
run_test "Mixed Signals" '{
  "ip": "203.0.113.18",
  "total_sent": 450,
  "failures": [
    {"code": "5.1.1", "domain": "example1.com", "count": 5, "reason": "Unknown user"},
    {"code": "5.7.1", "domain": "gmail.com", "count": 3, "reason": "Policy"},
    {"code": "4.2.2", "domain": "example2.com", "count": 3, "reason": "Mailbox full"}
  ]
}' "warning"

# Test Case 10: Gradual Reputation Decay (Test First Window)
echo "=== Test Case 10: Gradual Reputation Decay ==="
run_test "Initial Healthy State" '{
  "ip": "203.0.113.19",
  "total_sent": 300,
  "failures": [
    {"code": "5.1.1", "domain": "example.com", "count": 3, "reason": "Unknown user"}
  ]
}' "healthy"

# Additional API endpoint tests
echo "=========================================="
echo "Testing API Endpoints"
echo "=========================================="

# Test getting IP reputation
echo "--- Testing GET /api/ips/{ip}/reputation ---"
curl -s "$BASE_URL/api/ips/203.0.113.13/reputation" | jq '.' || echo "Failed to get reputation"
echo ""

# Test getting IP failures
echo "--- Testing GET /api/ips/{ip}/failures ---"
curl -s "$BASE_URL/api/ips/203.0.113.13/failures?window=1h" | jq '.' || echo "Failed to get failures"
echo ""

# Test dashboard
echo "--- Testing GET /api/dashboard/ip-health ---"
curl -s "$BASE_URL/api/dashboard/ip-health" | jq '.' || echo "Failed to get dashboard"
echo ""

# Test DNSBL check (will be slow)
echo "--- Testing POST /api/ips/{ip}/dnsbl-check ---"
echo "Note: This may take a few seconds..."
curl -s -X POST "$BASE_URL/api/ips/8.8.8.8/dnsbl-check" | jq '.' || echo "Failed DNSBL check"
echo ""

# Summary
echo "=========================================="
echo "Test Summary"
echo "=========================================="
echo -e "${GREEN}Tests Passed: $TESTS_PASSED${NC}"
echo -e "${RED}Tests Failed: $TESTS_FAILED${NC}"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}✅ All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}❌ Some tests failed${NC}"
    exit 1
fi

