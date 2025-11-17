#!/bin/bash

echo "=================================="
echo "🚀 STARTING IP REPUTATION SYSTEM"
echo "=================================="
echo ""

# Navigate to project directory
cd /Users/Mounir/Task-Master/Codebase/golang-backend-service

# Step 1: Start Docker Desktop if not running
echo "Step 1: Checking Docker..."
if ! docker ps > /dev/null 2>&1; then
    echo "⚠️  Docker is not running. Starting Docker Desktop..."
    open -a Docker
    echo "Waiting 30 seconds for Docker to start..."
    sleep 30
fi

# Verify Docker is ready
echo "Verifying Docker is ready..."
for i in {1..10}; do
    if docker ps > /dev/null 2>&1; then
        echo "✅ Docker is running!"
        break
    fi
    echo "Waiting for Docker... ($i/10)"
    sleep 3
done

if ! docker ps > /dev/null 2>&1; then
    echo "❌ Docker failed to start. Please start Docker Desktop manually and run this script again."
    exit 1
fi

# Step 2: Start backend services
echo ""
echo "Step 2: Starting backend services (Go app + PostgreSQL)..."
echo "Note: This will create a fresh database with the latest schema"
docker compose -f Context/Data/docker-compose.yml up --build -d

# Wait for services to be ready
echo "Waiting for services to start..."
sleep 10

# Check if app is healthy
echo "Checking backend health..."
for i in {1..15}; do
    if curl -s http://127.0.0.1:8080/health > /dev/null 2>&1; then
        echo "✅ Backend is healthy!"
        break
    fi
    echo "Waiting for backend... ($i/15)"
    sleep 2
done

# Step 3: Start web server for dashboard
echo ""
echo "Step 3: Starting web server for test dashboard..."
cd web
python3 -m http.server 8888 > /dev/null 2>&1 &
WEB_SERVER_PID=$!
cd ..

sleep 2

echo ""
echo "=================================="
echo "✅ EVERYTHING IS RUNNING!"
echo "=================================="
echo ""
echo "🌐 Services Available:"
echo "   • Backend API:       http://localhost:8080"
echo "   • Health Check:      http://localhost:8080/health"
echo "   • Swagger UI:        http://localhost:8080/swagger/index.html"
echo "   • Test Dashboard:    http://localhost:8888/test-dashboard.html"
echo "   • Prometheus:        http://localhost:9090"
echo "   • Grafana:           http://localhost:3000"
echo ""
echo "📊 Opening Test Dashboard..."
sleep 2
open http://localhost:8888/test-dashboard.html

echo ""
echo "=================================="
echo "🧪 READY TO TEST!"
echo "=================================="
echo ""
echo "In the dashboard that just opened:"
echo "   1. Click 'Run All Tests' button"
echo "   2. Watch the results appear"
echo ""
echo "Or run tests from command line:"
echo "   ./scripts/test-ip-reputation.sh"
echo ""
echo "To stop everything:"
echo "   ./STOP_EVERYTHING.sh"
echo ""

