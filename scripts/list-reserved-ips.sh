#!/bin/bash

# List reserved IPs
# This script lists all reserved IPs with optional filtering

set -e

# Configuration
API_URL="${API_URL:-http://localhost:8080}"
ENDPOINT="/api/v1/ips/reserved"
STATUS="${1:-}"  # Optional: reserved, in_use, released, quarantined
BLACKLISTED="${2:-}"  # Optional: true, false

echo "========================================="
echo "List Reserved IPs"
echo "========================================="
echo ""

# Build query parameters
QUERY_PARAMS=""
if [ -n "$STATUS" ]; then
    QUERY_PARAMS="?status=$STATUS"
fi
if [ -n "$BLACKLISTED" ]; then
    if [ -z "$QUERY_PARAMS" ]; then
        QUERY_PARAMS="?blacklisted=$BLACKLISTED"
    else
        QUERY_PARAMS="${QUERY_PARAMS}&blacklisted=$BLACKLISTED"
    fi
fi

echo "Filters:"
[ -n "$STATUS" ] && echo "  Status: $STATUS" || echo "  Status: All"
[ -n "$BLACKLISTED" ] && echo "  Blacklisted: $BLACKLISTED" || echo "  Blacklisted: All"
echo ""

# Make the request
response=$(curl -s -X GET "${API_URL}${ENDPOINT}${QUERY_PARAMS}" \
    -H "Content-Type: application/json" \
    -w "\n%{http_code}")

# Extract HTTP status code from last line
http_code=$(echo "$response" | tail -n 1)
body=$(echo "$response" | sed '$d')

if [ "$http_code" -eq 200 ]; then
    echo "✓ Retrieved reserved IPs"
    echo ""
    
    if command -v jq &> /dev/null; then
        count=$(echo "$body" | jq -r '.count')
        echo "Total: $count IPs"
        echo ""
        
        # Display IPs in a table format
        echo "ID    IP Address          Status       Location    Blacklisted  Reserved At"
        echo "----  ------------------  -----------  ----------  -----------  --------------------------"
        echo "$body" | jq -r '.ips[] | "\(.id)\t\(.ip_address)\t\(.status)\t\(.location)\t\(.is_blacklisted)\t\(.reserved_at)"' | column -t -s $'\t'
    else
        echo "$body"
    fi
else
    echo "✗ Failed to retrieve reserved IPs"
    echo ""
    echo "Response:"
    echo "$body"
    exit 1
fi

echo ""
echo "========================================="
echo ""
echo "Usage:"
echo "  $0                    # List all IPs"
echo "  $0 reserved           # List only reserved IPs"
echo "  $0 in_use             # List only in-use IPs"
echo "  $0 reserved false     # List reserved non-blacklisted IPs"
echo "========================================="

