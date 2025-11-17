#!/bin/bash

# Filter and search logs by level or keyword

echo "🔍 Log Filter Tool"
echo "=================="
echo ""
echo "Usage examples:"
echo "  ./filter-logs.sh error       # Show only errors"
echo "  ./filter-logs.sh info        # Show only info logs"
echo "  ./filter-logs.sh user        # Search for 'user' keyword"
echo ""

# Use Docker from application bundle
DOCKER="/Applications/Docker.app/Contents/Resources/bin/docker"

# Navigate to project root (parent of scripts directory)
cd "$(dirname "$0")/.."

if [ -z "$1" ]; then
    echo "Please provide a filter term (error, info, warn, or any keyword)"
    exit 1
fi

FILTER=$1

echo "Filtering logs for: $FILTER"
echo "Press Ctrl+C to stop"
echo ""

$DOCKER compose -f Context/Data/docker-compose.yml logs -f app | grep -i "$FILTER" --color=always

