package service

import (
	"context"
	"fmt"
	"github.com/gofiber/fiber/v2/log"
	"github.com/sam-berry/ecfr-analyzer/server/concurrent"
	"github.com/sam-berry/ecfr-analyzer/server/dao"
	"github.com/sam-berry/ecfr-analyzer/server/data"
	"github.com/sam-berry/ecfr-analyzer/server/parser"
)

type CfrStructureService struct {
	TitleDAO         *dao.TitleDAO
	CfrStructureDAO  *dao.CfrStructureDAO
}

// ProcessAllTitles parses and stores the CFR structure for all titles
func (s *CfrStructureService) ProcessAllTitles(
	ctx context.Context,
	titlesFilter []string,
) error {
	s.logInfo("Start")

	// Get all titles
	titles, err := s.TitleDAO.FindAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to find titles: %w", err)
	}

	// Apply filter if provided
	if len(titlesFilter) > 0 {
		filterMap := make(map[string]bool)
		for _, t := range titlesFilter {
			filterMap[t] = true
		}

		var filteredTitles []*data.Title
		for _, title := range titles {
			if filterMap[fmt.Sprintf("%d", title.Name)] {
				filteredTitles = append(filteredTitles, title)
			}
		}
		titles = filteredTitles
	}

	s.logInfo(fmt.Sprintf("Processing %d titles", len(titles)))

	// Create concurrent runner with limited concurrency
	runner := concurrent.NewRunner[*data.Title, string](concurrent.RunnerConfig{
		MaxConcurrency: 5, // Process 5 titles concurrently
		LogPrefix:      "CFR Structure Parser",
	})

	// Process titles concurrently
	result := runner.Run(titles, func(
		title *data.Title,
		messages chan<- string,
		results chan<- string,
		errors chan<- error,
	) {
		messages <- fmt.Sprintf("Processing: Title %d", title.Name)

		err := s.processTitle(ctx, title)
		if err != nil {
			messages <- fmt.Sprintf("Failed: Title %d - %v", title.Name, err)
			errors <- fmt.Errorf("title %d: %w", title.Name, err)
			return
		}

		messages <- fmt.Sprintf("Success: Title %d", title.Name)
		results <- fmt.Sprintf("Title %d", title.Name)
	})

	if len(result.Errors) > 0 {
		s.logInfo(fmt.Sprintf("Completed with %d errors", len(result.Errors)))
		for _, err := range result.Errors {
			s.logInfo(fmt.Sprintf("Error: %v", err))
		}
	} else {
		s.logInfo(fmt.Sprintf("Successfully processed %d titles", len(result.Results)))
	}

	s.logInfo("Complete")
	return nil
}

// processTitle parses and stores the CFR structure for a single title
func (s *CfrStructureService) processTitle(
	ctx context.Context,
	title *data.Title,
) error {
	// Get the XML content
	xmlContent, err := s.TitleDAO.GetContent(ctx, title.Name)
	if err != nil {
		return fmt.Errorf("failed to get title content: %w", err)
	}

	// Parse the XML
	cfrParser := parser.NewCfrParser(title.InternalId, title.Name)
	parseResult, err := cfrParser.Parse(xmlContent)
	if err != nil {
		return fmt.Errorf("failed to parse XML: %w", err)
	}

	// Delete existing structures for this title (if any)
	err = s.CfrStructureDAO.DeleteByTitleId(ctx, title.InternalId)
	if err != nil {
		return fmt.Errorf("failed to delete existing structures: %w", err)
	}

	// Store the parsed structures
	if len(parseResult.Structures) > 0 {
		// Build parent-child relationships
		// First pass: create a map of path to structure for quick lookup
		pathMap := make(map[string]*data.CfrStructure)
		for _, structure := range parseResult.Structures {
			pathMap[structure.Path] = structure
		}

		// Second pass: set parent IDs based on path hierarchy
		for _, structure := range parseResult.Structures {
			// Find parent path by removing the last segment
			parentPath := getParentPath(structure.Path)
			if parentPath != "" {
				if parent, ok := pathMap[parentPath]; ok {
					structure.ParentId = &parent.InternalId
				}
			}
		}

		err = s.CfrStructureDAO.BatchInsert(ctx, parseResult.Structures)
		if err != nil {
			return fmt.Errorf("failed to insert structures: %w", err)
		}
	}

	return nil
}

// getParentPath extracts the parent path from a hierarchical path
// e.g., "1/3/A/1" -> "1/3/A"
func getParentPath(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[:i]
		}
	}
	return ""
}

func (s *CfrStructureService) logInfo(message string) {
	log.Info(fmt.Sprintf("CFR Structure Process: %v", message))
}
