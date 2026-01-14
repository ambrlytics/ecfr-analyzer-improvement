#!/bin/bash
set -e

echo "🎨 Starting CFR Analyzer UI..."
echo "================================"
echo "Development server will start on http://localhost:3000"
echo ""
echo "Press Ctrl+C to stop"
echo ""

cd ui
npm run dev
