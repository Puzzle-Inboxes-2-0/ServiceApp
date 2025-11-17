cl#!/bin/bash

# Verification Script for 15-Test Suite Expansion
# This script verifies that all 15 tests are properly configured and working

set -e

BASE_URL="${1:-http://127.0.0.1:8080}"
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "======================================"
echo "15-Test Suite Verification"
echo "======================================"
echo ""

# Function to print colored output
print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}ℹ️  $1${NC}"
}

# Check if service is running
echo "Step 1: Checking if service is running..."
if curl -s "$BASE_URL/health" > /dev/null 2>&1; then
    print_success "Service is running at $BASE_URL"
else
    print_error "Service is not running at $BASE_URL"
    echo "Please start the service first:"
    echo "  docker compose -f Context/Data/docker-compose.yml up -d"
    exit 1
fi
echo ""

# Get test cases
echo "Step 2: Fetching test cases..."
TEST_CASES=$(curl -s "$BASE_URL/api/testing/test-cases")
TEST_COUNT=$(echo "$TEST_CASES" | jq '. | length')

if [ "$TEST_COUNT" -eq 15 ]; then
    print_success "Found $TEST_COUNT test cases (expected 15)"
else
    print_error "Expected 15 test cases, found $TEST_COUNT"
    exit 1
fi
echo ""

# Verify all test IDs exist
echo "Step 3: Verifying all test IDs (test-1 through test-15)..."
for i in {1..15}; do
    TEST_ID="test-$i"
    if echo "$TEST_CASES" | jq -e ".[] | select(.id == \"$TEST_ID\")" > /dev/null; then
        echo "  ✓ $TEST_ID exists"
    else
        print_error "$TEST_ID not found"
        exit 1
    fi
done
print_success "All test IDs present"
echo ""

# Verify new tests (11-15) exist
echo "Step 4: Verifying NEW test cases (test-11 through test-15)..."
NEW_TESTS=("test-11" "test-12" "test-13" "test-14" "test-15")
NEW_TEST_NAMES=(
    "Microsoft Reputation Block - 5.7.606"
    "Content Spam Detection - 5.7.512"
    "Infrastructure Issues - Multiple DNS Problems"
    "DKIM/ARC Authentication Failure - 5.7.26"
    "Policy Rejections - Temporary Issues"
)

for i in "${!NEW_TESTS[@]}"; do
    TEST_ID="${NEW_TESTS[$i]}"
    TEST_NAME="${NEW_TEST_NAMES[$i]}"
    if echo "$TEST_CASES" | jq -e ".[] | select(.id == \"$TEST_ID\")" > /dev/null; then
        print_success "$TEST_ID: $TEST_NAME"
    else
        print_error "$TEST_ID not found"
        exit 1
    fi
done
echo ""

# Verify error code coverage
echo "Step 5: Verifying error code coverage..."
ERROR_CODES=(
    "5.7.1" "5.7.606" "5.7.512"  # PRIMARY
    "5.7.23" "5.7.26"             # AUTHENTICATION
    "5.7.25" "5.7.27" "5.7.7" "5.1.8"  # INFRASTRUCTURE
    "4.7.0" "4.7.1" "5.7.510"     # POLICY
    "5.1.1" "4.2.2" "5.4.1"       # OTHER
)

echo "Checking that all 15 error codes are represented in test cases..."
for CODE in "${ERROR_CODES[@]}"; do
    if echo "$TEST_CASES" | jq -e ".[] | .failures[] | select(.code == \"$CODE\")" > /dev/null 2>&1; then
        echo "  ✓ $CODE is tested"
    else
        print_error "$CODE is not tested"
        exit 1
    fi
done
print_success "All 15 error codes covered"
echo ""

# Run the full test suite
echo "Step 6: Running full test suite..."
print_info "This may take 10-30 seconds..."
RESULT=$(curl -s -X POST "$BASE_URL/api/testing/test-suite/run")
TOTAL=$(echo "$RESULT" | jq -r '.total_tests')
PASSED=$(echo "$RESULT" | jq -r '.passed_tests')
FAILED=$(echo "$RESULT" | jq -r '.failed_tests')
EXEC_TIME=$(echo "$RESULT" | jq -r '.execution_time_ms')

echo ""
echo "Test Results:"
echo "  Total Tests:     $TOTAL"
echo "  Passed Tests:    $PASSED"
echo "  Failed Tests:    $FAILED"
echo "  Execution Time:  ${EXEC_TIME}ms"
echo ""

if [ "$TOTAL" -eq 15 ] && [ "$PASSED" -eq 15 ] && [ "$FAILED" -eq 0 ]; then
    print_success "All 15 tests passed!"
else
    print_error "Some tests failed. Expected: 15/15, Got: $PASSED/$TOTAL"
    echo ""
    echo "Failed tests:"
    echo "$RESULT" | jq -r '.results[] | select(.passed == false) | "\(.test_id): \(.test_name) - Expected: \(.expected_status), Got: \(.actual_status)"'
    exit 1
fi
echo ""

# Verify status level coverage
echo "Step 7: Verifying status level coverage..."
HEALTHY=$(echo "$RESULT" | jq '[.results[] | select(.expected_status == "healthy")] | length')
WARNING=$(echo "$RESULT" | jq '[.results[] | select(.expected_status == "warning")] | length')
QUARANTINE=$(echo "$RESULT" | jq '[.results[] | select(.expected_status == "quarantine")] | length')
BLACKLISTED=$(echo "$RESULT" | jq '[.results[] | select(.expected_status == "blacklisted")] | length')

echo "  Healthy:      $HEALTHY tests"
echo "  Warning:      $WARNING tests"
echo "  Quarantine:   $QUARANTINE tests"
echo "  Blacklisted:  $BLACKLISTED tests"

if [ $HEALTHY -gt 0 ] && [ $WARNING -gt 0 ] && [ $QUARANTINE -gt 0 ] && [ $BLACKLISTED -gt 0 ]; then
    print_success "All status levels covered"
else
    print_error "Not all status levels are covered"
    exit 1
fi
echo ""

# Final summary
echo "======================================"
echo "VERIFICATION COMPLETE"
echo "======================================"
echo ""
print_success "All verifications passed! ✨"
echo ""
echo "Summary:"
echo "  ✅ Service is running"
echo "  ✅ 15 test cases found"
echo "  ✅ All test IDs present (test-1 through test-15)"
echo "  ✅ 5 new tests added (test-11 through test-15)"
echo "  ✅ All 15 error codes covered"
echo "  ✅ All 15 tests passed"
echo "  ✅ All status levels tested"
echo ""
echo "Error Code Coverage:"
echo "  • PRIMARY:         3/3 codes (100%)"
echo "  • AUTHENTICATION:  2/2 codes (100%)"
echo "  • INFRASTRUCTURE:  4/4 codes (100%)"
echo "  • POLICY:          3/3 codes (100%)"
echo "  • OTHER:           3/3 codes (100%)"
echo "  • TOTAL:          15/15 codes (100%)"
echo ""
echo "Next steps:"
echo "  1. Open web dashboard: open web/test-dashboard.html"
echo "  2. View test coverage: cat docs/TEST_COVERAGE.md"
echo "  3. Read changelog: cat docs/CHANGELOG_TEST_EXPANSION.md"
echo ""
print_success "🎉 Test suite expansion successful!"

