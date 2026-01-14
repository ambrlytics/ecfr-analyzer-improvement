package data

import "time"

// TitleVersion represents a historical version of a CFR title
// Used for tracking changes over time
type TitleVersion struct {
	InternalId    int       `json:"-"`
	Id            string    `json:"id"`
	TitleId       int       `json:"titleId"`
	TitleNumber   int       `json:"titleNumber"`
	VersionDate   time.Time `json:"versionDate"`   // The date this version was effective
	CreatedAt     time.Time `json:"createdAt"`
}

// TitleVersionWithContent extends TitleVersion to include the XML content
// Used when fetching full version data for processing
type TitleVersionWithContent struct {
	TitleVersion
	Content string `json:"content"` // XML content
}
