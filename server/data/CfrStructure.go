package data

import "time"

// CfrStructure represents a hierarchical element in the CFR XML structure
// DIV1-DIV9 elements with their metadata and content
type CfrStructure struct {
	InternalId    int       `json:"-"`
	Id            string    `json:"id"`
	TitleId       int       `json:"titleId"`
	TitleNumber   int       `json:"titleNumber"`
	DivType       string    `json:"divType"`       // TITLE, SUBTITLE, CHAPTER, SUBCHAP, PART, SUBPART, SUBJGRP, SECTION, APPENDIX
	DivLevel      int       `json:"divLevel"`      // 1-9
	Identifier    string    `json:"identifier"`    // N attribute value
	NodeId        *string   `json:"nodeId"`        // NODE attribute value (optional)
	Heading       *string   `json:"heading"`       // HEAD element content (optional)
	TextContent   *string   `json:"textContent"`   // Full text content (optional)
	WordCount     int       `json:"wordCount"`     // Precomputed word count
	ParentId      *int      `json:"parentId"`      // Parent structure element (optional for root)
	Path          string    `json:"path"`          // Hierarchical path (e.g., "1/3/A/1")
	CreatedAt     time.Time `json:"createdAt"`
}

// DivType constants for structured CFR elements
const (
	DivTypeTitle     = "TITLE"
	DivTypeSubtitle  = "SUBTITLE"
	DivTypeChapter   = "CHAPTER"
	DivTypeSubchap   = "SUBCHAP"
	DivTypePart      = "PART"
	DivTypeSubpart   = "SUBPART"
	DivTypeSubjgrp   = "SUBJGRP"
	DivTypeSection   = "SECTION"
	DivTypeAppendix  = "APPENDIX"
)
