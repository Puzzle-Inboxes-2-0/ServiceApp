#!/bin/bash

# Pretty Log Viewer for Logrus JSON logs

echo "🔍 GoLang Backend Service - Live Log Viewer"
echo "=========================================="
echo ""
echo "Viewing logs from data-app-1..."
echo "Press Ctrl+C to stop"
echo ""

# Use Docker from application bundle
DOCKER="/Applications/Docker.app/Contents/Resources/bin/docker"

# Navigate to project root (parent of scripts directory)
cd "$(dirname "$0")/.."

# Follow logs and pretty-print JSON
$DOCKER compose -f Context/Data/docker-compose.yml logs -f app | while read line; do
    # Try to extract and pretty-print JSON if it exists
    json=$(echo "$line" | grep -oP '\{.*\}' || echo "")
    
    if [ ! -z "$json" ]; then
        echo "$json" | jq -C '.' 2>/dev/null || echo "$line"
    else
        echo "$line"
    fi
done

