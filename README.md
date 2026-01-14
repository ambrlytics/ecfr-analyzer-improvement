# CFR-Metrics.com - Federal Regulations Metric Analyzer

## Overview

The purpose of this application is to download Federal Regulations data and produce insights based on the text and
information available. Data is sourced from the [ECFR Bulk Data Repository](https://www.govinfo.gov/bulkdata/ECFR) and
the [eCFR API](https://www.ecfr.gov/developers/documentation/api/v1#/).

The general approach that this application takes is to define jobs which handle the heavy lifting of downloading,
parsing, and collecting insights from the large set of CFR data. These programs then store computed values so that they
can have constant lookup for display and analysis purposes.

This system is currently running on [cfr-metrics.com](https://cfr-metrics.com). 

### Data Model

The following tables make up the data model for this application:

* `agency`: Stores agency data fetched from
  the [/admin/v1/agencies.json](https://www.ecfr.gov/developers/documentation/api/v1#/) API
* `title`: Stores title XML downloaded from the [ECFR Bulk Data Repository](https://www.govinfo.gov/bulkdata/ECFR)
* `computed_value`: A key-value store for computed metrics
* `cfr_structure`: Stores the hierarchical structure of CFR documents (DIV1-DIV9 elements) with precomputed text values for efficient querying
* `title_version`: Stores historical versions of CFR titles for change tracking over time

[Source](https://github.com/sam-berry/ecfr-analyzer/blob/main/server/sql/ecfr_analyzer.sql)

### Server Architecture

There is a single server definition which handles all API requests - both publicly available endpoints, and the
authenticated import endpoints. It is a Go server which is intended to be run in a serverless environment via
Dockerfile.

[Source](https://github.com/sam-berry/ecfr-analyzer/tree/main/server)

### UI Architecture

The UI for [cfr-metrics.com](https://cfr-metrics.com) is built using NextJS with an emphasis on SSR-capable pages which
can be easily cached. The app is intended to be run in a serverless environment via Dockerfile.

[Source](https://github.com/sam-berry/ecfr-analyzer/tree/main/ui)

### Cloud Architecture

All infrastructure that powers [cfr-metrics.com](https://cfr-metrics.com) is running in Google Cloud via serverless
architecture. This includes:

* Cloud Run Services for both the UI and Server applications
* Cloud CDN to cache UI assets and artifacts
* Cloud SQL using Postgres as a backend
* Load balancing, routing, and SSL

## Data Population Workflow

Assuming the application is running, these are the steps to download and populate the data needed to
power [cfr-metrics.com](https://cfr-metrics.com), from scratch:

**`URL_ROOT`**: Locally this will be `http://localhost:8090`. For production it is `https://cfr-metrics.com`.
**`TOKEN`**: This value is set by the `ECFR_ADMIN_TOKEN` environment variable.

### Step 1: Import Agencies

To download and save all agencies, run:

```
curl -X POST -H 'Authorization: Bearer TOKEN' 'URL_ROOT/ecfr-service/import-agencies'
```

### Step 2: Import Titles

To download and save all current titles, run:

```
curl -X POST -H 'Authorization: Bearer TOKEN' 'URL_ROOT/ecfr-service/import-titles'
```

### Step 3: Compute Title Metrics

To process titles and compute metrics, run:

```
curl -X POST -H 'Authorization: Bearer TOKEN' 'URL_ROOT/ecfr-service/compute/title-metrics'
```

### Step 4: Compute Agency Metrics

To process metrics for all agencies, run:

```
curl -X POST -H 'Authorization: Bearer TOKEN' 'URL_ROOT/ecfr-service/compute/agency-metrics'
```

### Step 5: Compute Sub-Agency Metrics

To process metrics for all sub-agencies, run:

```
curl -X POST -H 'Authorization: Bearer TOKEN' 'URL_ROOT/ecfr-service/compute/sub-agency-metrics'
```

### Step 6 (Optional): Parse CFR Structure

To parse and store the hierarchical structure of CFR documents for efficient querying:

```
curl -X POST -H 'Authorization: Bearer TOKEN' 'URL_ROOT/ecfr-service/parse/cfr-structure'
```

This will process all titles and extract chapters, parts, sections, etc. as structured data. You can also filter specific titles:

```
curl -X POST -H 'Authorization: Bearer TOKEN' 'URL_ROOT/ecfr-service/parse/cfr-structure?titles=1,2,3'
```

### Step 7 (Optional): Import Historical Titles

To import historical CFR title versions for change tracking:

```
curl -X POST -H 'Authorization: Bearer TOKEN' 'URL_ROOT/ecfr-service/import/historical-titles?date=2024-01-01'
```

You can also filter specific titles:

```
curl -X POST -H 'Authorization: Bearer TOKEN' 'URL_ROOT/ecfr-service/import/historical-titles?date=2024-01-01&titles=1,2,3'
```

### Step 8 (Optional): Compute Changes Between Dates

To compute and store metrics about changes between two versions:

```
curl -X POST -H 'Authorization: Bearer TOKEN' 'URL_ROOT/ecfr-service/compute/changes?startDate=2024-01-01&endDate=2024-12-31'
```

These steps will generate all of the data needed to power the UI with constant lookup times.

## Development Setup

The following technologies are required:

* Go 1.21+
* Postgres 17
* Node 22

### Environment Variables

```
export ECFR_ADMIN_TOKEN="any token or UUID"
export ECFR_DB_USER="ecfr-app"
export ECFR_DB_PASS=""
export ECFR_DB_HOST="localhost"
export ECFR_DB_PORT="5432"
export ECFR_DB_NAME="ecfr"
export ECFR_DB_INSTANCE_CONNECTION_NAME=""
export ECFR_DEVELOPMENT="true"
```

### Setup Database

1. `createuser ecfr-app`
2. `createdb ecfr`
3. `psql ecfr`
4. `grant all privileges on database ecfr to "ecfr-app";`
5. `grant all on schema public TO "ecfr-app";`
6. Run statements in [ecfr_analyzer.sql](https://github.com/sam-berry/ecfr-analyzer/blob/main/server/sql/ecfr_analyzer.sql)
7. Run migration scripts in `server/sql/migrations/` for new features:
   - `001_add_cfr_structure.sql` - Adds structured CFR data table
   - `002_add_title_version.sql` - Adds historical title version tracking

### Run Server

1. `cd /server`
2. `go run server.go`

### Run UI

1. `cd /ui`
2. `npm install`
3. `npm run dev`

## Find Agencies That Are Missing Computed Values

When computing agency metrics, it can be useful to run the following query to see if any agencies were missed. EPA,
Treasury, and Agriculture occasionally timeout and should be checked with this.

```
SELECT a.slug, cv.id
FROM agency a
         LEFT JOIN computed_value cv ON cv.key = CONCAT('agency-metrics__', a.agencyId)
WHERE cv.id IS NULL;
```

Failed agencies can be run individually, or in bulk via the [`import-specific-agencies.sh`](https://github.com/sam-berry/ecfr-analyzer/blob/main/server/scripts/import-specific-agencies.sh) script.

## Recent Improvements

### Structured CFR Data
The application now parses CFR XML documents and stores the hierarchical structure (DIV1-DIV9 elements) as structured data in the `cfr_structure` table. This enables:
- Efficient querying of chapters, parts, sections, and other CFR elements without XPath operations
- Precomputed word counts for each structural element
- Fast lookups by hierarchical path or element type

### Common Goroutine Runner
A reusable concurrent processing utility (`concurrent.Runner`) has been implemented to standardize goroutine, channel, and wait group patterns throughout the codebase. This provides:
- Configurable concurrency limits
- Consistent error handling and logging
- Simplified concurrent processing in services

### Refactored Sub-Agency Logic
The sub-agency metrics computation has been refactored to eliminate the `onlySubAgencies` flag parameter. The new `ComputedValueServiceRefactored` provides:
- Separate methods for agency and sub-agency processing (`ProcessAgencyMetrics` and `ProcessSubAgencyMetrics`)
- Cleaner separation of concerns
- Easier to understand and maintain code

### Historical CFR Tracking
The application now supports importing and tracking historical versions of CFR titles with:
- `TitleVersionService` for importing historical data
- `ChangeTrackingService` for computing metrics about changes over time
- APIs to query changes between dates and generate change reports
- Analysis of word count and section count changes

### New API Endpoints

**CFR Structure:**
- `POST /ecfr-service/parse/cfr-structure` - Parse and store CFR hierarchical structure

**Historical Titles:**
- `POST /ecfr-service/import/historical-titles` - Import historical title versions

**Change Tracking:**
- `POST /ecfr-service/compute/changes` - Compute changes between dates
- `GET /ecfr-service/changes/summary` - Get change summary for date range
- `GET /ecfr-service/changes/top` - Get titles with most significant changes
- `GET /ecfr-service/changes/report` - Generate human-readable change report
