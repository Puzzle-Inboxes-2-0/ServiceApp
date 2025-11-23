#!/bin/bash

# Check IONOS quota
# This script displays current IONOS quota usage

set -e

# Configuration
API_URL="${API_URL:-http://localhost:8080}"
ENDPOINT="/api/v1/ips/quota"

echo "========================================="
echo "IONOS Quota Check"
echo "========================================="
echo ""

# Make the quota check request
response=$(curl -s -X GET "${API_URL}${ENDPOINT}" \
    -H "Content-Type: application/json" \
    -w "\n%{http_code}")

# Extract HTTP status code from last line
http_code=$(echo "$response" | tail -n 1)
body=$(echo "$response" | sed '$d')

if [ "$http_code" -eq 200 ]; then
    echo "✓ Quota check successful"
    echo ""
    
    # Parse and display quota information
    if command -v jq &> /dev/null; then
        total=$(echo "$body" | jq -r '.total_blocks')
        protected=$(echo "$body" | jq -r '.protected_blocks')
        single=$(echo "$body" | jq -r '.single_ip_blocks')
        limit=$(echo "$body" | jq -r '.estimated_limit')
        remaining=$(echo "$body" | jq -r '.remaining')
        
        echo "Quota Usage:"
        echo "  Total Blocks:      $total"
        echo "  Protected Blocks:  $protected (11-IP blocks - never deleted)"
        echo "  Single IP Blocks:  $single"
        echo "  Estimated Limit:   $limit"
        echo "  Remaining:         $remaining"
    else
        echo "Response:"
        echo "$body"
    fi
else
    echo "✗ Quota check failed"
    echo ""
    echo "Response:"
    echo "$body"
    exit 1
fi

echo ""
echo "========================================="

