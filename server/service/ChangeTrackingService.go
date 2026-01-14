package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v2/log"
	"github.com/sam-berry/ecfr-analyzer/server/dao"
	"github.com/sam-berry/ecfr-analyzer/server/data"
	"github.com/sam-berry/ecfr-analyzer/server/parser"
	"strings"
	"time"
)

type ChangeTrackingService struct {
	TitleVersionDAO  *dao.TitleVersionDAO
	ComputedValueDAO *dao.ComputedValueDAO
	TitleDAO         *dao.TitleDAO
}

// TitleChange represents changes in a title between two versions
type TitleChange struct {
	TitleNumber         int       `json:"titleNumber"`
	StartDate           time.Time `json:"startDate"`
	EndDate             time.Time `json:"endDate"`
	WordCountChange     int       `json:"wordCountChange"`     // Positive = added, negative = removed
	SectionCountChange  int       `json:"sectionCountChange"`  // Positive = added, negative = removed
	TotalWordsStart     int       `json:"totalWordsStart"`
	TotalWordsEnd       int       `json:"totalWordsEnd"`
	TotalSectionsStart  int       `json:"totalSectionsStart"`
	TotalSectionsEnd    int       `json:"totalSectionsEnd"`
	PercentWordChange   float64   `json:"percentWordChange"`
	PercentSectionChange float64  `json:"percentSectionChange"`
}

// ComputeChangesForDateRange computes changes for all titles between two dates
func (s *ChangeTrackingService) ComputeChangesForDateRange(
	ctx context.Context,
	startDate time.Time,
	endDate time.Time,
	titlesFilter []string,
) error {
	s.logInfo(fmt.Sprintf("Computing changes from %s to %s",
		startDate.Format("2006-01-02"),
		endDate.Format("2006-01-02")))

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

	var allChanges []TitleChange

	for _, title := range titles {
		change, err := s.computeTitleChange(ctx, title.Name, startDate, endDate)
		if err != nil {
			s.logInfo(fmt.Sprintf("Failed to compute change for title %d: %v", title.Name, err))
			continue
		}

		allChanges = append(allChanges, *change)
		s.logInfo(fmt.Sprintf("Title %d: %d words changed, %d sections changed",
			title.Name,
			change.WordCountChange,
			change.SectionCountChange))
	}

	// Store the computed changes
	changeBytes, err := json.Marshal(allChanges)
	if err != nil {
		return fmt.Errorf("failed to marshal changes: %w", err)
	}

	cv := &data.ComputedValue{
		Key:  fmt.Sprintf("title-changes__%s__%s",
			startDate.Format("2006-01-02"),
			endDate.Format("2006-01-02")),
		Data: changeBytes,
	}

	err = s.ComputedValueDAO.Insert(ctx, cv)
	if err != nil {
		return fmt.Errorf("failed to store changes: %w", err)
	}

	s.logInfo(fmt.Sprintf("Successfully computed changes for %d titles", len(allChanges)))
	return nil
}

// computeTitleChange computes the change for a single title between two dates
func (s *ChangeTrackingService) computeTitleChange(
	ctx context.Context,
	titleNumber int,
	startDate time.Time,
	endDate time.Time,
) (*TitleChange, error) {
	// Get version for start date
	startVersion, err := s.TitleVersionDAO.GetContentByVersion(ctx, titleNumber, startDate)
	if err != nil || startVersion == nil {
		return nil, fmt.Errorf("failed to get start version: %w", err)
	}

	// Get version for end date
	endVersion, err := s.TitleVersionDAO.GetContentByVersion(ctx, titleNumber, endDate)
	if err != nil || endVersion == nil {
		return nil, fmt.Errorf("failed to get end version: %w", err)
	}

	// Parse both versions
	startMetrics, err := s.parseVersionMetrics(startVersion.TitleId, titleNumber, startVersion.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse start version: %w", err)
	}

	endMetrics, err := s.parseVersionMetrics(endVersion.TitleId, titleNumber, endVersion.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse end version: %w", err)
	}

	// Compute changes
	wordChange := endMetrics.TotalWords - startMetrics.TotalWords
	sectionChange := endMetrics.TotalSections - startMetrics.TotalSections

	// Compute percentages
	var percentWordChange float64
	if startMetrics.TotalWords > 0 {
		percentWordChange = float64(wordChange) / float64(startMetrics.TotalWords) * 100
	}

	var percentSectionChange float64
	if startMetrics.TotalSections > 0 {
		percentSectionChange = float64(sectionChange) / float64(startMetrics.TotalSections) * 100
	}

	return &TitleChange{
		TitleNumber:          titleNumber,
		StartDate:            startDate,
		EndDate:              endDate,
		WordCountChange:      wordChange,
		SectionCountChange:   sectionChange,
		TotalWordsStart:      startMetrics.TotalWords,
		TotalWordsEnd:        endMetrics.TotalWords,
		TotalSectionsStart:   startMetrics.TotalSections,
		TotalSectionsEnd:     endMetrics.TotalSections,
		PercentWordChange:    percentWordChange,
		PercentSectionChange: percentSectionChange,
	}, nil
}

// VersionMetrics holds metrics for a specific version
type VersionMetrics struct {
	TotalWords    int
	TotalSections int
}

// parseVersionMetrics parses a version and extracts metrics
func (s *ChangeTrackingService) parseVersionMetrics(
	titleId int,
	titleNumber int,
	content string,
) (*VersionMetrics, error) {
	cfrParser := parser.NewCfrParser(titleId, titleNumber)
	parseResult, err := cfrParser.Parse(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse version: %w", err)
	}

	// Count sections (DIV8 elements)
	sectionCount := 0
	for _, structure := range parseResult.Structures {
		if structure.DivType == data.DivTypeSection {
			sectionCount++
		}
	}

	return &VersionMetrics{
		TotalWords:    parseResult.TotalWords,
		TotalSections: sectionCount,
	}, nil
}

// GetChangeSummary retrieves a summary of changes across all titles for a date range
func (s *ChangeTrackingService) GetChangeSummary(
	ctx context.Context,
	startDate time.Time,
	endDate time.Time,
) ([]TitleChange, error) {
	key := fmt.Sprintf("title-changes__%s__%s",
		startDate.Format("2006-01-02"),
		endDate.Format("2006-01-02"))

	cv, err := s.ComputedValueDAO.FindByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to find changes: %w", err)
	}

	if cv == nil {
		return nil, fmt.Errorf("no changes found for date range")
	}

	var changes []TitleChange
	err = json.Unmarshal(cv.Data, &changes)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal changes: %w", err)
	}

	return changes, nil
}

// GetTopChangingTitles returns the titles with the most significant changes
func (s *ChangeTrackingService) GetTopChangingTitles(
	ctx context.Context,
	startDate time.Time,
	endDate time.Time,
	limit int,
) ([]TitleChange, error) {
	changes, err := s.GetChangeSummary(ctx, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// Sort by absolute word count change
	sortedChanges := make([]TitleChange, len(changes))
	copy(sortedChanges, changes)

	// Simple bubble sort for top N (good enough for small datasets)
	for i := 0; i < len(sortedChanges)-1; i++ {
		for j := 0; j < len(sortedChanges)-i-1; j++ {
			if abs(sortedChanges[j].WordCountChange) < abs(sortedChanges[j+1].WordCountChange) {
				sortedChanges[j], sortedChanges[j+1] = sortedChanges[j+1], sortedChanges[j]
			}
		}
	}

	// Return top N
	if limit > len(sortedChanges) {
		limit = len(sortedChanges)
	}

	return sortedChanges[:limit], nil
}

// GenerateChangeReport generates a human-readable report of changes
func (s *ChangeTrackingService) GenerateChangeReport(
	ctx context.Context,
	startDate time.Time,
	endDate time.Time,
) (string, error) {
	changes, err := s.GetChangeSummary(ctx, startDate, endDate)
	if err != nil {
		return "", err
	}

	var report strings.Builder
	report.WriteString(fmt.Sprintf("CFR Change Report: %s to %s\n\n",
		startDate.Format("2006-01-02"),
		endDate.Format("2006-01-02")))

	totalWordChange := 0
	totalSectionChange := 0

	for _, change := range changes {
		totalWordChange += change.WordCountChange
		totalSectionChange += change.SectionCountChange

		report.WriteString(fmt.Sprintf("Title %d:\n", change.TitleNumber))
		report.WriteString(fmt.Sprintf("  Words: %d -> %d (change: %+d, %.2f%%)\n",
			change.TotalWordsStart,
			change.TotalWordsEnd,
			change.WordCountChange,
			change.PercentWordChange))
		report.WriteString(fmt.Sprintf("  Sections: %d -> %d (change: %+d, %.2f%%)\n\n",
			change.TotalSectionsStart,
			change.TotalSectionsEnd,
			change.SectionCountChange,
			change.PercentSectionChange))
	}

	report.WriteString(fmt.Sprintf("Total across all titles:\n"))
	report.WriteString(fmt.Sprintf("  Word change: %+d\n", totalWordChange))
	report.WriteString(fmt.Sprintf("  Section change: %+d\n", totalSectionChange))

	return report.String(), nil
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func (s *ChangeTrackingService) logInfo(message string) {
	log.Info(fmt.Sprintf("Change Tracking Process: %v", message))
}
