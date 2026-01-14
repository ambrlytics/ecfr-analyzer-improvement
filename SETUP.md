# CFR Analyzer - Development Environment Setup Guide

This guide walks you through setting up your local development environment for the CFR Analyzer project.

## ✅ Prerequisites

You need the following installed:
- **Go 1.21+** (you have: 1.24.7 ✅)
- **PostgreSQL 16+** (you have: 16.11 ✅)
- **Node 22+** (you have: 22.21.1 ✅)

## 🚀 Quick Start

### 1. Set Up Environment Variables

```bash
# Copy the example environment file
cp .env.example .env.local

# Edit .env.local and set your ECFR_ADMIN_TOKEN to a unique value
# Then load the variables:
source .env.local
```

This sets up all required environment variables including your admin token.

### 2. Start PostgreSQL

Make sure PostgreSQL is running:

```bash
# Check status
pg_isready

# Start if needed (Linux)
sudo systemctl start postgresql

# Or on macOS
brew services start postgresql@16
```

### 3. Set Up Database

Create the database user, database, and run migrations:

```bash
# Create user and database
sudo -u postgres createuser ecfr-app
sudo -u postgres createdb ecfr

# Grant permissions
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE ecfr TO \"ecfr-app\";"
sudo -u postgres psql ecfr -c "GRANT ALL ON SCHEMA public TO \"ecfr-app\";"

# Run the setup script (creates tables and runs migrations)
./setup-database.sh
```

**Alternative manual setup:**
```bash
# Run base schema
sudo -u postgres psql -U postgres -d ecfr -f server/sql/ecfr_analyzer.sql

# Run migrations for new features
sudo -u postgres psql -U postgres -d ecfr -f server/sql/migrations/001_add_cfr_structure.sql
sudo -u postgres psql -U postgres -d ecfr -f server/sql/migrations/002_add_title_version.sql
```

### 4. Verify Database Setup

```bash
psql -U ecfr-app -d ecfr -c '\dt'
```

You should see these tables:
- `agency`
- `title`
- `computed_value`
- `cfr_structure` (new)
- `title_version` (new)

### 5. Install Dependencies

#### Server Dependencies
```bash
cd server
go mod download
cd ..
```

#### UI Dependencies
```bash
cd ui
npm install
cd ..
```

### 6. Start Development Servers

Open two terminal windows:

**Terminal 1 - Backend Server:**
```bash
./dev-server.sh
```
Server will start on http://localhost:8090

**Terminal 2 - Frontend UI:**
```bash
./dev-ui.sh
```
UI will start on http://localhost:3000

## 🧪 Testing the Setup

### Test 1: Health Check
```bash
curl http://localhost:8090/health
```

### Test 2: Import Sample Data

First, import agencies:
```bash
curl -X POST \
  -H "Authorization: Bearer $ECFR_ADMIN_TOKEN" \
  http://localhost:8090/ecfr-service/import-agencies
```

Then import a small title (Title 1):
```bash
curl -X POST \
  -H "Authorization: Bearer $ECFR_ADMIN_TOKEN" \
  'http://localhost:8090/ecfr-service/import-titles?titles=1'
```

### Test 3: Parse CFR Structure (New Feature!)
```bash
curl -X POST \
  -H "Authorization: Bearer $ECFR_ADMIN_TOKEN" \
  'http://localhost:8090/ecfr-service/parse/cfr-structure?titles=1'
```

### Test 4: Compute Metrics
```bash
# Title metrics
curl -X POST \
  -H "Authorization: Bearer $ECFR_ADMIN_TOKEN" \
  http://localhost:8090/ecfr-service/compute/title-metrics

# Agency metrics (filter to avoid timeout)
curl -X POST \
  -H "Authorization: Bearer $ECFR_ADMIN_TOKEN" \
  'http://localhost:8090/ecfr-service/compute/agency-metrics?agencies=environmental-protection-agency'
```

## 📁 Project Structure

```
ecfr-analyzer-improvement/
├── .env.local              # Environment variables (source this!)
├── setup-database.sh       # Database setup script
├── dev-server.sh          # Start backend server
├── dev-ui.sh              # Start frontend UI
├── README.md              # Main documentation
├── server/                # Go backend
│   ├── api/               # HTTP endpoints
│   ├── concurrent/        # NEW: Goroutine runner utility
│   ├── dao/               # Database access layer
│   ├── data/              # Data models
│   ├── parser/            # NEW: CFR XML parser
│   ├── service/           # Business logic
│   ├── sql/               # Database schemas
│   │   └── migrations/    # NEW: Database migrations
│   └── server.go          # Main entry point
└── ui/                    # Next.js frontend
    ├── src/
    └── package.json
```

## 🔑 Environment Variables

Your `.env.local` file contains:

| Variable | Value | Description |
|----------|-------|-------------|
| `ECFR_ADMIN_TOKEN` | `dev-token-local-1234567890` | Token for admin endpoints |
| `ECFR_DB_USER` | `ecfr-app` | Database username |
| `ECFR_DB_PASS` | (empty) | Database password |
| `ECFR_DB_HOST` | `localhost` | Database host |
| `ECFR_DB_PORT` | `5432` | Database port |
| `ECFR_DB_NAME` | `ecfr` | Database name |
| `ECFR_DEVELOPMENT` | `true` | Development mode flag |

## 🆕 New Features Available

### 1. Structured CFR Data
Parse CFR XML into structured database records:
```bash
POST /ecfr-service/parse/cfr-structure?titles=1,2,3
```

### 2. Historical Title Tracking
Import historical versions:
```bash
POST /ecfr-service/import/historical-titles?date=2024-01-01&titles=1
```

### 3. Change Tracking
Compute changes between dates:
```bash
POST /ecfr-service/compute/changes?startDate=2024-01-01&endDate=2024-12-31
GET /ecfr-service/changes/summary?startDate=2024-01-01&endDate=2024-12-31
```

## 🐛 Troubleshooting

### PostgreSQL Connection Errors
```bash
# Check if PostgreSQL is running
pg_isready

# Check if you can connect
psql -U ecfr-app -d ecfr -c '\l'
```

### Port Already in Use
If port 8090 or 3000 is already in use:
```bash
# Find process using port
lsof -i :8090
lsof -i :3000

# Kill the process
kill -9 <PID>
```

### Go Module Issues
```bash
cd server
go mod tidy
go mod download
```

### Node Module Issues
```bash
cd ui
rm -rf node_modules package-lock.json
npm install
```

## 📚 Next Steps

1. **Read the README.md** for architecture details
2. **Explore the API** using the endpoints above
3. **Check the UI** at http://localhost:3000
4. **Review the code** in `server/` and `ui/` directories

## 💡 Development Tips

- **Hot Reload**: The Go server doesn't have hot reload by default. Use `air` or restart manually.
- **Frontend Hot Reload**: Next.js has built-in hot reload - changes appear automatically.
- **Database Changes**: After schema changes, restart the server.
- **Logs**: Server logs appear in Terminal 1, UI logs in Terminal 2.

## 🤝 Need Help?

- Check the [README.md](README.md) for detailed documentation
- Review API endpoints in `server/api/`
- Check service logic in `server/service/`
- Database schema in `server/sql/`

Happy coding! 🎉
