#!/bin/bash

# Reserve clean IPs from IONOS
# This script calls the IP reservation API endpoint

set -e

# Configuration
API_URL="${API_URL:-http://localhost:8080}"
ENDPOINT="/api/v1/ips/reserve"
COUNT="${1:-11}"  # Default to 11 IPs if not specified
LOCATION="${2:-us/ewr}"  # Default to Newark if not specified

echo "========================================="
echo "IONOS IP Reservation Script"
echo "========================================="
echo ""
echo "API URL: ${API_URL}"
echo "Endpoint: ${ENDPOINT}"
echo "Count: ${COUNT}"
echo "Location: ${LOCATION}"
echo ""

# Validate count
if ! [[ "$COUNT" =~ ^[0-9]+$ ]] || [ "$COUNT" -lt 1 ] || [ "$COUNT" -gt 50 ]; then
    echo "Error: Count must be a number between 1 and 50"
    exit 1
fi

# Make the reservation request
echo "Starting IP reservation process..."
echo "This may take several minutes depending on the number of IPs requested."
echo ""

response=$(curl -s -X POST "${API_URL}${ENDPOINT}" \
    -H "Content-Type: application/json" \
    -d "{\"count\": ${COUNT}, \"location\": \"${LOCATION}\"}" \
    -w "\n%{http_code}")

# Extract HTTP status code from last line
http_code=$(echo "$response" | tail -n 1)
body=$(echo "$response" | sed '$d')

echo ""
echo "HTTP Status: ${http_code}"

if [ "$http_code" -eq 201 ]; then
    echo "✓ IP reservation completed"
    echo ""
    echo "Response:"
    echo "$body" | jq '.' 2>/dev/null || echo "$body"
    
    # Extract and display reserved IPs
    reserved_ips=$(echo "$body" | jq -r '.reserved_ips[]?.ip_address' 2>/dev/null || echo "")
    if [ -n "$reserved_ips" ]; then
        echo ""
        echo "Reserved IPs:"
        echo "$reserved_ips" | while read ip; do
            echo "  - $ip"
        done
    fi
else
    echo "✗ IP reservation failed"
    echo ""
    echo "Response:"
    echo "$body"
    exit 1
fi

echo ""
echo "========================================="
echo "Reservation script completed"
echo "========================================="

