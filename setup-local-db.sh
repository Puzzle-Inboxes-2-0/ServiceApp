#!/bin/bash

# Setup database in your LOCAL PostgreSQL 17
# This script creates the mydb database and sets up tables

echo "========================================="
echo "Setting up LOCAL PostgreSQL Database"
echo "========================================="
echo ""

# We'll use your local PostgreSQL on port 5432
# You need to provide the password

echo "Creating database 'mydb' (will skip if exists)..."
createdb -h localhost -p 5432 -U postgres mydb 2>/dev/null || echo "Database mydb already exists"

echo ""
echo "Setting up tables from init.sql..."
psql -h localhost -p 5432 -U postgres -d mydb -f Context/Data/init.sql

echo ""
echo "✓ Database setup complete!"
echo ""
echo "Database credentials:"
echo "  Host: localhost"
echo "  Port: 5432"
echo "  User: postgres"
echo "  Database: mydb"
echo ""
echo "Test connection:"
echo "  psql -h localhost -p 5432 -U postgres -d mydb"
echo ""
echo "========================================="

