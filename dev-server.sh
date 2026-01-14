#!/bin/bash
set -e

# Load environment variables
source .env.local

echo "🚀 Starting CFR Analyzer Server..."
echo "=================================="
echo "Environment: DEVELOPMENT"
echo "Port: 8090"
echo "Admin Token: ${ECFR_ADMIN_TOKEN}"
echo ""
echo "Press Ctrl+C to stop"
echo ""

cd server
go run server.go
