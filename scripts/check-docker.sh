#!/bin/bash

# Helper script to check if Docker is ready

echo "🔍 Checking Docker status..."
echo ""

# Use Docker from application bundle
DOCKER="/Applications/Docker.app/Contents/Resources/bin/docker"

# Check if Docker CLI exists
if [ ! -f "$DOCKER" ]; then
    echo "❌ Docker CLI not found at: $DOCKER"
    echo ""
    echo "Is Docker Desktop installed?"
    echo "  Download from: https://www.docker.com/products/docker-desktop"
    exit 1
fi

echo "✅ Docker CLI found"

# Check if Docker daemon is running
if $DOCKER info > /dev/null 2>&1; then
    echo "✅ Docker daemon is running"
    echo ""
    $DOCKER version
    echo ""
    echo "🎉 Docker is ready! You can now run:"
    echo "  ./scripts/start-services.sh"
else
    echo "❌ Docker daemon is NOT running"
    echo ""
    echo "Please check Docker Desktop:"
    echo "  1. Look for the Docker whale icon in your menu bar"
    echo "  2. If you don't see it, open Docker Desktop from Applications"
    echo "  3. Wait for the icon to become solid (not animated)"
    echo "  4. Then run this script again"
    echo ""
    exit 1
fi

