#!/bin/bash

# API Testing Script for GoLang Backend Service
# Works with both localhost and IP address

# Determine which host to use
if curl -s --connect-timeout 2 http://localhost:8080/health > /dev/null 2>&1; then
    HOST="localhost"
    echo "✓ Using localhost"
else
    HOST="127.0.0.1"
    echo "✓ Using 127.0.0.1 (localhost not available)"
fi

BASE_URL="http://${HOST}:8080"

echo ""
echo "========================================="
echo "GoLang Backend Service API Tests"
echo "Base URL: ${BASE_URL}"
echo "========================================="
echo ""

# Test 1: Health Check
echo "1. Testing Health Check..."
curl -s "${BASE_URL}/health" | jq . || echo "Failed"
echo ""

# Test 2: Get All Users
echo "2. Testing Get All Users..."
curl -s "${BASE_URL}/users" | jq . || echo "Failed"
echo ""

# Test 3: Create New User
echo "3. Testing Create New User..."
NEW_USER=$(curl -s -X POST -H "Content-Type: application/json" \
  -d '{"username":"testuser_'$(date +%s)'", "email":"test_'$(date +%s)'@example.com"}' \
  "${BASE_URL}/users")
echo "$NEW_USER" | jq .
USER_ID=$(echo "$NEW_USER" | jq -r '.id')
echo ""

# Test 4: Get User by ID
if [ ! -z "$USER_ID" ] && [ "$USER_ID" != "null" ]; then
    echo "4. Testing Get User by ID (ID: $USER_ID)..."
    curl -s "${BASE_URL}/users/${USER_ID}" | jq . || echo "Failed"
    echo ""
else
    echo "4. Skipping Get User by ID (no user created)"
    echo ""
fi

# Test 5: Metrics Endpoint
echo "5. Testing Metrics Endpoint (first 20 lines)..."
curl -s "${BASE_URL}/metrics" | head -20
echo "..."
echo ""

# Summary
echo "========================================="
echo "Test Summary"
echo "========================================="
echo "Swagger UI: ${BASE_URL}/swagger/index.html"
echo "Metrics: ${BASE_URL}/metrics"
echo "Health: ${BASE_URL}/health"
echo ""
echo "To view logs:"
echo "  docker-compose -f Context/Data/docker-compose.yml logs -f app"
echo ""
echo "To stop service:"
echo "  docker-compose -f Context/Data/docker-compose.yml down"
echo "========================================="

