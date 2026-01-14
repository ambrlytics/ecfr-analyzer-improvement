package parser

import (
	"encoding/xml"
	"fmt"
	"github.com/sam-berry/ecfr-analyzer/server/data"
	"io"
	"strings"
)

// XMLDiv represents a DIV element in the CFR XML structure
type XMLDiv struct {
	XMLName  xml.Name  `xml:""`
	Type     string    `xml:"TYPE,attr"`
	N        string    `xml:"N,attr"`
	Node     string    `xml:"NODE,attr"`
	Head     string    `xml:"HEAD"`
	Content  string    `xml:",innerxml"`
	Children []XMLDiv  `xml:",any"`
}

// CfrParser parses CFR XML documents into structured data
type CfrParser struct {
	titleId     int
	titleNumber int
}

// NewCfrParser creates a new CFR parser
func NewCfrParser(titleId int, titleNumber int) *CfrParser {
	return &CfrParser{
		titleId:     titleId,
		titleNumber: titleNumber,
	}
}

// ParseResult contains the parsed CFR structure elements
type ParseResult struct {
	Structures []*data.CfrStructure
	TotalWords int
}

// Parse parses the CFR XML content and extracts the hierarchical structure
func (p *CfrParser) Parse(xmlContent string) (*ParseResult, error) {
	decoder := xml.NewDecoder(strings.NewReader(xmlContent))

	var structures []*data.CfrStructure
	var totalWords int

	// Parse the XML document
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error parsing XML: %w", err)
		}

		if startElement, ok := token.(xml.StartElement); ok {
			// Check if this is a DIV element
			if strings.HasPrefix(startElement.Name.Local, "DIV") && len(startElement.Name.Local) == 4 {
				divLevel := int(startElement.Name.Local[3] - '0') // Extract number from DIV1-DIV9

				// Parse this DIV element and its children
				divStructures, words := p.parseDivElement(decoder, &startElement, divLevel, nil, "")
				structures = append(structures, divStructures...)
				totalWords += words
			}
		}
	}

	return &ParseResult{
		Structures: structures,
		TotalWords: totalWords,
	}, nil
}

// parseDivElement recursively parses a DIV element and its children
func (p *CfrParser) parseDivElement(
	decoder *xml.Decoder,
	startElement *xml.StartElement,
	divLevel int,
	parentId *int,
	parentPath string,
) ([]*data.CfrStructure, int) {

	// Extract attributes
	var divType string
	var identifier string
	var nodeId *string

	for _, attr := range startElement.Attr {
		switch attr.Name.Local {
		case "TYPE":
			divType = attr.Value
		case "N":
			identifier = attr.Value
		case "NODE":
			nodeId = &attr.Value
		}
	}

	// Build path
	path := parentPath
	if path != "" {
		path += "/"
	}
	path += identifier

	// Parse the content of this element
	var heading *string
	var textContent strings.Builder
	var childStructures []*data.CfrStructure
	var inHead bool

	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}

		// Check for end of this DIV element
		if endElement, ok := token.(xml.EndElement); ok {
			if endElement.Name.Local == startElement.Name.Local {
				break
			}
		}

		// Handle start elements
		if childStart, ok := token.(xml.StartElement); ok {
			if childStart.Name.Local == "HEAD" {
				inHead = true
				// Read the HEAD content
				headText := ""
				for {
					headToken, err := decoder.Token()
					if err != nil {
						break
					}
					if headEnd, ok := headToken.(xml.EndElement); ok && headEnd.Name.Local == "HEAD" {
						break
					}
					if charData, ok := headToken.(xml.CharData); ok {
						headText += string(charData)
					}
				}
				headText = strings.TrimSpace(headText)
				heading = &headText
				inHead = false
			} else if strings.HasPrefix(childStart.Name.Local, "DIV") && len(childStart.Name.Local) == 4 {
				// This is a child DIV element
				childDivLevel := int(childStart.Name.Local[3] - '0')

				// We'll need to assign parent_id after we create the current structure
				// For now, parse with nil parent and we'll update it later
				childDivs, _ := p.parseDivElement(decoder, &childStart, childDivLevel, nil, path)
				childStructures = append(childStructures, childDivs...)
			} else {
				// Other elements - extract text content
				p.extractTextContent(decoder, &childStart, &textContent)
			}
		}

		// Handle character data
		if charData, ok := token.(xml.CharData); ok && !inHead {
			text := strings.TrimSpace(string(charData))
			if text != "" {
				textContent.WriteString(text)
				textContent.WriteString(" ")
			}
		}
	}

	// Build the structure object
	text := strings.TrimSpace(textContent.String())
	wordCount := countWords(text)

	var textPtr *string
	if text != "" {
		textPtr = &text
	}

	structure := &data.CfrStructure{
		TitleId:     p.titleId,
		TitleNumber: p.titleNumber,
		DivType:     divType,
		DivLevel:    divLevel,
		Identifier:  identifier,
		NodeId:      nodeId,
		Heading:     heading,
		TextContent: textPtr,
		WordCount:   wordCount,
		ParentId:    parentId,
		Path:        path,
	}

	// Combine current structure with children
	structures := []*data.CfrStructure{structure}

	// Update child structures to reference this parent
	// Note: This assumes we process structures in order and can use array indices
	// In practice, we'd need to assign InternalId after database insertion
	for _, child := range childStructures {
		// We'll set parent references properly in the service layer after DB insertion
		structures = append(structures, child)
	}

	totalWords := wordCount
	for _, child := range childStructures {
		totalWords += child.WordCount
	}

	return structures, totalWords
}

// extractTextContent recursively extracts text content from an element
func (p *CfrParser) extractTextContent(
	decoder *xml.Decoder,
	startElement *xml.StartElement,
	textContent *strings.Builder,
) {
	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}

		if endElement, ok := token.(xml.EndElement); ok {
			if endElement.Name.Local == startElement.Name.Local {
				break
			}
		}

		if childStart, ok := token.(xml.StartElement); ok {
			p.extractTextContent(decoder, &childStart, textContent)
		}

		if charData, ok := token.(xml.CharData); ok {
			text := strings.TrimSpace(string(charData))
			if text != "" {
				textContent.WriteString(text)
				textContent.WriteString(" ")
			}
		}
	}
}

// countWords counts the number of words in a text string
func countWords(text string) int {
	if text == "" {
		return 0
	}

	words := strings.Fields(text)
	return len(words)
}

// GetDivTypeForLevel returns the typical DIV type for a given level
// Note: This is based on common CFR structure, but actual TYPE attributes should be used
func GetDivTypeForLevel(level int) string {
	switch level {
	case 1:
		return data.DivTypeTitle
	case 2:
		return data.DivTypeSubtitle
	case 3:
		return data.DivTypeChapter
	case 4:
		return data.DivTypeSubchap
	case 5:
		return data.DivTypePart
	case 6:
		return data.DivTypeSubpart
	case 7:
		return data.DivTypeSubjgrp
	case 8:
		return data.DivTypeSection
	case 9:
		return data.DivTypeAppendix
	default:
		return ""
	}
}
