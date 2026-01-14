-- Migration: Add structured CFR data table
-- This table stores the hierarchical structure of CFR elements (DIV1-DIV9)
-- allowing for efficient querying without XPath operations

CREATE TABLE cfr_structure
(
    id                SERIAL PRIMARY KEY,
    structure_id      UUID UNIQUE NOT NULL,
    title_id          INTEGER     NOT NULL REFERENCES title (id) ON DELETE CASCADE,
    title_number      INTEGER     NOT NULL,
    div_type          TEXT        NOT NULL, -- TITLE, SUBTITLE, CHAPTER, SUBCHAP, PART, SUBPART, SUBJGRP, SECTION, APPENDIX
    div_level         INTEGER     NOT NULL, -- 1-9
    identifier        TEXT        NOT NULL, -- The N attribute value (e.g., "1", "A", "§ 1.1")
    node_id           TEXT,                 -- The NODE attribute value (internal tracking)
    heading           TEXT,                 -- The HEAD element content
    text_content      TEXT,                 -- Full text content of this element
    word_count        INTEGER     NOT NULL DEFAULT 0,
    parent_id         INTEGER REFERENCES cfr_structure (id) ON DELETE CASCADE,
    path              TEXT        NOT NULL, -- Hierarchical path (e.g., "1/3/A/1" for quick queries)
    created_timestamp TIMESTAMP   NOT NULL DEFAULT NOW()
);

-- Indexes for efficient querying
CREATE INDEX idx_cfr_structure_title_id ON cfr_structure (title_id);
CREATE INDEX idx_cfr_structure_title_number ON cfr_structure (title_number);
CREATE INDEX idx_cfr_structure_div_type ON cfr_structure (div_type);
CREATE INDEX idx_cfr_structure_parent_id ON cfr_structure (parent_id);
CREATE INDEX idx_cfr_structure_path ON cfr_structure (path);
CREATE INDEX idx_cfr_structure_identifier ON cfr_structure (identifier);

-- Composite index for common queries
CREATE INDEX idx_cfr_structure_title_div ON cfr_structure (title_number, div_type);
