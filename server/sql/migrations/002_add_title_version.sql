-- Migration: Add title version tracking for historical CFR data
-- This table stores historical versions of CFR titles to enable change tracking over time

CREATE TABLE title_version
(
    id                SERIAL PRIMARY KEY,
    version_id        UUID UNIQUE    NOT NULL,
    title_id          INTEGER        NOT NULL REFERENCES title (id) ON DELETE CASCADE,
    title_number      INTEGER        NOT NULL,
    content           XML            NOT NULL,
    version_date      DATE           NOT NULL, -- The date this version was effective
    created_timestamp TIMESTAMP      NOT NULL DEFAULT NOW(),
    UNIQUE (title_number, version_date) -- Only one version per title per date
);

-- Indexes for efficient querying
CREATE INDEX idx_title_version_title_id ON title_version (title_id);
CREATE INDEX idx_title_version_title_number ON title_version (title_number);
CREATE INDEX idx_title_version_date ON title_version (version_date);

-- Composite index for date range queries
CREATE INDEX idx_title_version_title_date ON title_version (title_number, version_date DESC);
