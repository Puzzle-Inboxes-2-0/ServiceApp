#!/bin/bash

# Start the GoLang backend service

echo "🐳 Starting GoLang Backend Service..."
echo ""

# Use Docker from application bundle
DOCKER="/Applications/Docker.app/Contents/Resources/bin/docker"

# Check if Docker daemon is running
if ! $DOCKER info > /dev/null 2>&1; then
    echo "❌ Docker daemon is not running!"
    echo "Please start Docker Desktop and try again."
    exit 1
fi

echo "✅ Docker daemon is running"
echo ""

# Navigate to project root (parent of scripts directory)
cd "$(dirname "$0")/.."

# Start services
echo "🚀 Building and starting services..."
$DOCKER compose -f Context/Data/docker-compose.yml up --build -d

if [ $? -eq 0 ]; then
    echo ""
    echo "✅ Services started successfully!"
    echo ""
    echo "Waiting for services to be ready..."
    sleep 5
    echo ""
    echo "📊 Service Status:"
    $DOCKER compose -f Context/Data/docker-compose.yml ps
    echo ""
    echo "🌐 Access Points:"
    echo "  - API Base URL: http://localhost:8080"
    echo "  - Health Check: http://localhost:8080/health"
    echo "  - Swagger UI: http://localhost:8080/swagger/index.html"
    echo "  - Metrics: http://localhost:8080/metrics"
    echo "  - Database: localhost:5433"
    echo ""
    echo "📝 Useful Commands:"
    echo "  - Run tests: ./scripts/test-api.sh"
    echo "  - View logs: $DOCKER compose -f Context/Data/docker-compose.yml logs -f app"
    echo "  - Stop services: $DOCKER compose -f Context/Data/docker-compose.yml down"
    echo ""
else
    echo ""
    echo "❌ Failed to start services"
    exit 1
fi

