#!/bin/bash

# Start the GoLang backend with full monitoring stack

echo "🚀 Starting GoLang Backend with Full Monitoring Stack"
echo "======================================================="
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

# Start services with monitoring stack
echo "🚀 Starting services with Prometheus, Grafana, and Loki..."
$DOCKER compose -f Context/Data/docker-compose.monitoring.yml up -d

if [ $? -eq 0 ]; then
    echo ""
    echo "✅ All services started successfully!"
    echo ""
    echo "Waiting for services to be ready..."
    sleep 10
    echo ""
    echo "📊 Monitoring Stack Access:"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "  🌐 Your Application:"
    echo "     - API: http://localhost:8080"
    echo "     - Swagger: http://localhost:8080/swagger/index.html"
    echo "     - Metrics: http://localhost:8080/metrics"
    echo ""
    echo "  📊 Prometheus (Metrics Database):"
    echo "     - URL: http://localhost:9090"
    echo "     - Query your metrics"
    echo "     - View targets and alerts"
    echo ""
    echo "  📈 Grafana (Visualization Dashboard):"
    echo "     - URL: http://localhost:3000"
    echo "     - Username: admin"
    echo "     - Password: admin"
    echo "     - Pre-configured with Prometheus & Loki"
    echo ""
    echo "  📝 Loki (Log Aggregation):"
    echo "     - URL: http://localhost:3100"
    echo "     - Access logs through Grafana"
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "🎯 Quick Start Guide:"
    echo ""
    echo "  1. Open Grafana: http://localhost:3000"
    echo "  2. Login with admin/admin (change password when prompted)"
    echo "  3. Go to: http://localhost:3000/dashboard/import"
    echo "  4. Type Dashboard ID: 10826 (Go Metrics)"
    echo "  5. Click Load, then click Import (Prometheus is already default!)"
    echo ""
    echo "📝 Useful Commands:"
    echo "  - View logs: $DOCKER compose -f Context/Data/docker-compose.monitoring.yml logs -f app"
    echo "  - Stop all: $DOCKER compose -f Context/Data/docker-compose.monitoring.yml down"
    echo "  - Check status: $DOCKER compose -f Context/Data/docker-compose.monitoring.yml ps"
    echo ""
else
    echo ""
    echo "❌ Failed to start services"
    exit 1
fi

