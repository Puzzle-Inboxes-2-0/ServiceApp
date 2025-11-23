#!/bin/bash

# Cleanup unused IONOS IP blocks
# This script calls the cleanup API endpoint to remove single-IP blocks not in use

set -e

# Configuration
API_URL="${API_URL:-http://localhost:8080}"
ENDPOINT="/api/v1/ips/cleanup"

echo "========================================="
echo "IONOS IP Block Cleanup Script"
echo "========================================="
echo ""
echo "API URL: ${API_URL}"
echo "Endpoint: ${ENDPOINT}"
echo ""

# Make the cleanup request
echo "Initiating cleanup..."
response=$(curl -s -X POST "${API_URL}${ENDPOINT}" \
    -H "Content-Type: application/json" \
    -w "\n%{http_code}")

# Extract HTTP status code from last line
http_code=$(echo "$response" | tail -n 1)
body=$(echo "$response" | sed '$d')

echo ""
echo "HTTP Status: ${http_code}"

if [ "$http_code" -eq 200 ]; then
    echo "✓ Cleanup completed successfully"
    echo ""
    echo "Response:"
    echo "$body" | jq '.' 2>/dev/null || echo "$body"
else
    echo "✗ Cleanup failed"
    echo ""
    echo "Response:"
    echo "$body"
    exit 1
fi

echo ""
echo "========================================="
echo "Cleanup script completed"
echo "========================================="

