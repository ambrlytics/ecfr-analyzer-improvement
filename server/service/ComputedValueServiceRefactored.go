package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v2/log"
	"github.com/sam-berry/ecfr-analyzer/server/concurrent"
	"github.com/sam-berry/ecfr-analyzer/server/dao"
	"github.com/sam-berry/ecfr-analyzer/server/data"
	"strings"
)

type ComputedValueServiceRefactored struct {
	TitleMetricService  *TitleMetricService
	AgencyMetricService *AgencyMetricService
	ComputedValueDAO    *dao.ComputedValueDAO
	AgencyDAO           *dao.AgencyDAO
}

func (s *ComputedValueServiceRefactored) ProcessTitleMetrics(
	ctx context.Context,
) error {
	result, err := s.TitleMetricService.CountAllWordsAndSections(ctx)
	if err != nil {
		return fmt.Errorf("failed to count title metrics, %w", err)
	}

	rBytes, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal title metrics, %w", err)
	}

	cv := &data.ComputedValue{
		Key:  data.ComputedValueKeyGlobalTitleMetrics(),
		Data: rBytes,
	}

	err = s.ComputedValueDAO.Insert(ctx, cv)
	if err != nil {
		return fmt.Errorf("failed to insert computed value, %w", err)
	}

	return nil
}

// ProcessAgencyMetrics processes metrics for parent agencies
func (s *ComputedValueServiceRefactored) ProcessAgencyMetrics(
	ctx context.Context,
	agenciesFilter []string,
) error {
	s.logInfo("Start - Agency Metrics")

	agencies, err := s.getFilteredAgencies(ctx, agenciesFilter)
	if err != nil {
		return err
	}

	// Create concurrent runner with limited concurrency
	runner := concurrent.NewRunner[*data.Agency, string](concurrent.RunnerConfig{
		MaxConcurrency: 3,
		LogPrefix:      "Agency Metrics",
	})

	// Process agencies concurrently
	result := runner.Run(agencies, func(
		agency *data.Agency,
		messages chan<- string,
		results chan<- string,
		errors chan<- error,
	) {
		s.processAgencyMetric(ctx, agency, messages, results, errors)
	})

	s.logResults("Agency Metrics", result.Results, result.Errors)
	return nil
}

// ProcessSubAgencyMetrics processes metrics for sub-agencies
func (s *ComputedValueServiceRefactored) ProcessSubAgencyMetrics(
	ctx context.Context,
) error {
	s.logInfo("Start - Sub-Agency Metrics")

	// Get all agencies and extract sub-agencies
	allAgencies, err := s.AgencyDAO.FindAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to find agencies, %w", err)
	}

	subAgencies := s.extractSubAgencies(allAgencies)
	s.logInfo(fmt.Sprintf("Processing %d sub-agencies", len(subAgencies)))

	// Create concurrent runner with limited concurrency
	runner := concurrent.NewRunner[*data.Agency, string](concurrent.RunnerConfig{
		MaxConcurrency: 3,
		LogPrefix:      "Sub-Agency Metrics",
	})

	// Process sub-agencies concurrently
	result := runner.Run(subAgencies, func(
		subAgency *data.Agency,
		messages chan<- string,
		results chan<- string,
		errors chan<- error,
	) {
		s.processSubAgencyMetric(ctx, subAgency, messages, results, errors)
	})

	s.logResults("Sub-Agency Metrics", result.Results, result.Errors)
	return nil
}

// processAgencyMetric processes metrics for a single parent agency
func (s *ComputedValueServiceRefactored) processAgencyMetric(
	ctx context.Context,
	agency *data.Agency,
	messages chan<- string,
	results chan<- string,
	errors chan<- error,
) {
	slug := agency.Slug
	messages <- fmt.Sprintf("Processing: %v", slug)

	// Count metrics for the agency
	result, err := s.AgencyMetricService.CountWordsAndSections(ctx, slug, "")
	if err != nil {
		messages <- fmt.Sprintf("failed to count agency metrics, %v, %v", slug, err)
		errors <- fmt.Errorf("agency %s: %w", slug, err)
		return
	}

	// Marshal results
	rBytes, err := json.Marshal(result)
	if err != nil {
		messages <- fmt.Sprintf("failed to marshall agency metrics, %v, %v", slug, err)
		errors <- fmt.Errorf("agency %s: %w", slug, err)
		return
	}

	// Store computed value
	cv := &data.ComputedValue{
		Key:  data.ComputedValueKeyAgencyMetric(agency.Id),
		Data: rBytes,
	}

	err = s.ComputedValueDAO.Insert(ctx, cv)
	if err != nil {
		messages <- fmt.Sprintf("failed to insert agency metrics, %v, %v", slug, err)
		errors <- fmt.Errorf("agency %s: %w", slug, err)
		return
	}

	messages <- fmt.Sprintf("Success: %v", slug)
	results <- slug
}

// processSubAgencyMetric processes metrics for a single sub-agency
func (s *ComputedValueServiceRefactored) processSubAgencyMetric(
	ctx context.Context,
	subAgency *data.Agency,
	messages chan<- string,
	results chan<- string,
	errors chan<- error,
) {
	if subAgency.Parent == nil {
		errors <- fmt.Errorf("sub-agency %s has no parent", subAgency.Name)
		return
	}

	parentSlug := subAgency.Parent.Slug
	subAgencyName := subAgency.Name

	messages <- fmt.Sprintf("Processing: %v (sub-agency of %v)", subAgencyName, parentSlug)

	// Count metrics for the sub-agency
	result, err := s.AgencyMetricService.CountWordsAndSections(ctx, parentSlug, subAgencyName)
	if err != nil {
		messages <- fmt.Sprintf("failed to count sub-agency metrics, %v, %v", subAgencyName, err)
		errors <- fmt.Errorf("sub-agency %s: %w", subAgencyName, err)
		return
	}

	// Marshal results
	rBytes, err := json.Marshal(result)
	if err != nil {
		messages <- fmt.Sprintf("failed to marshall sub-agency metrics, %v, %v", subAgencyName, err)
		errors <- fmt.Errorf("sub-agency %s: %w", subAgencyName, err)
		return
	}

	// Store computed value with sub-agency key
	cv := &data.ComputedValue{
		Key:  data.ComputedValueKeySubAgencyMetric(subAgency.Parent.Id, subAgencyName),
		Data: rBytes,
	}

	err = s.ComputedValueDAO.Insert(ctx, cv)
	if err != nil {
		messages <- fmt.Sprintf("failed to insert sub-agency metrics, %v, %v", subAgencyName, err)
		errors <- fmt.Errorf("sub-agency %s: %w", subAgencyName, err)
		return
	}

	messages <- fmt.Sprintf("Success: %v", subAgencyName)
	results <- subAgencyName
}

// getFilteredAgencies retrieves and filters agencies based on the provided filter
func (s *ComputedValueServiceRefactored) getFilteredAgencies(
	ctx context.Context,
	agenciesFilter []string,
) ([]*data.Agency, error) {
	agencies, err := s.AgencyDAO.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find agencies, %w", err)
	}

	if len(agenciesFilter) == 0 {
		return agencies, nil
	}

	// Build filter map
	filterMap := make(map[string]bool, len(agenciesFilter))
	for _, agency := range agenciesFilter {
		filterMap[agency] = true
	}

	// Filter agencies
	var filteredAgencies []*data.Agency
	for _, agency := range agencies {
		if filterMap[agency.Slug] {
			filteredAgencies = append(filteredAgencies, agency)
		}
	}

	return filteredAgencies, nil
}

// extractSubAgencies extracts all sub-agencies from the list of agencies
func (s *ComputedValueServiceRefactored) extractSubAgencies(
	agencies []*data.Agency,
) []*data.Agency {
	var subAgencies []*data.Agency

	for _, agency := range agencies {
		for _, child := range agency.Children {
			// Set parent reference
			child.Parent = agency
			subAgencies = append(subAgencies, child)
		}
	}

	return subAgencies
}

// logResults logs the results of a processing run
func (s *ComputedValueServiceRefactored) logResults(
	prefix string,
	results []string,
	errors []error,
) {
	if len(errors) > 0 {
		s.logInfo(fmt.Sprintf("%s - Completed with %d errors", prefix, len(errors)))
		for _, err := range errors {
			s.logInfo(fmt.Sprintf("%s - Error: %v", prefix, err))
		}
	}

	if len(results) > 0 {
		s.logInfo(fmt.Sprintf("%s - Successfully processed: %v", prefix, strings.Join(results, ", ")))
	}

	s.logInfo(fmt.Sprintf("%s - Complete", prefix))
}

func (s *ComputedValueServiceRefactored) logInfo(message string) {
	log.Info(fmt.Sprintf("Computed Value Process: %v", message))
}
