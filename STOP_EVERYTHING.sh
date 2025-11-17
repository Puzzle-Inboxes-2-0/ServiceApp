#!/bin/bash

echo "=================================="
echo "🛑 STOPPING IP REPUTATION SYSTEM"
echo "=================================="
echo ""

cd /Users/Mounir/Task-Master/Codebase/golang-backend-service

# Stop Docker containers
echo "Stopping Docker containers..."
docker compose -f Context/Data/docker-compose.yml down

# Kill web server
echo "Stopping web server..."
pkill -f "python3 -m http.server 8888"

echo ""
echo "✅ Everything stopped!"
echo ""

