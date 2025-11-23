#!/bin/bash

# Quick start script for IONOS IP Reservation Service
# This script sets up environment and starts the service

set -e

echo "========================================="
echo "Starting IONOS IP Reservation Service"
echo "========================================="
echo ""

# Load environment variables
if [ -f "scripts/setup-env.sh" ]; then
    echo "Loading environment variables..."
    source scripts/setup-env.sh
else
    echo "⚠️  Warning: scripts/setup-env.sh not found"
    echo "Please set IONOS_TOKEN manually:"
    echo "  export IONOS_TOKEN='your_token_here'"
    exit 1
fi

echo ""
echo "Checking dependencies..."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.23+ first."
    exit 1
fi

echo "✓ Go is installed: $(go version)"

# Check if database is accessible
echo ""
echo "Checking database connection..."
if command -v psql &> /dev/null; then
    if psql -h localhost -p 5432 -U postgres -d mydb -c "SELECT 1" &> /dev/null; then
        echo "✓ Database is accessible"
    else
        echo "⚠️  Warning: Cannot connect to database"
        echo "Make sure PostgreSQL is running:"
        echo "  cd Context/Data && docker-compose up -d"
    fi
else
    echo "⚠️  psql not found, skipping database check"
fi

echo ""
echo "Building and starting service..."
echo ""

# Run the service
go run cmd/server/main.go

