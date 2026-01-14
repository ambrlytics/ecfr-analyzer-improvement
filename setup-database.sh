#!/bin/bash
set -e

echo "🗄️  CFR Analyzer Database Setup"
echo "================================"

# Load environment variables
source .env.local

echo "Step 1: Creating database schema..."
sudo -u postgres psql -U postgres -d ecfr -f server/sql/ecfr_analyzer.sql

echo "Step 2: Running migrations..."
for migration in server/sql/migrations/*.sql; do
    echo "  Running $(basename $migration)..."
    sudo -u postgres psql -U postgres -d ecfr -f "$migration"
done

echo ""
echo "✅ Database setup complete!"
echo ""
echo "To verify, run:"
echo "  psql -U ecfr-app -d ecfr -c '\\dt'"
