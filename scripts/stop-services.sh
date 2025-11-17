#!/bin/bash

# Helper script to stop the GoLang backend service

echo "🛑 Stopping GoLang Backend Service..."
echo ""

# Use Docker from application bundle
DOCKER="/Applications/Docker.app/Contents/Resources/bin/docker"

# Navigate to project root (parent of scripts directory)
cd "$(dirname "$0")/.."

# Stop services
$DOCKER compose -f Context/Data/docker-compose.yml down

if [ $? -eq 0 ]; then
    echo ""
    echo "✅ Services stopped successfully!"
    echo ""
    echo "To remove all data (including database):"
    echo "  $DOCKER compose -f Context/Data/docker-compose.yml down -v"
    echo ""
else
    echo ""
    echo "❌ Failed to stop services"
    exit 1
fi

