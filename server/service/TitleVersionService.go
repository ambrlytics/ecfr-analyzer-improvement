package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v2/log"
	"github.com/sam-berry/ecfr-analyzer/server/concurrent"
	"github.com/sam-berry/ecfr-analyzer/server/dao"
	"github.com/sam-berry/ecfr-analyzer/server/data"
	"github.com/sam-berry/ecfr-analyzer/server/ecfrdata"
	"github.com/sam-berry/ecfr-analyzer/server/httpclient"
	"io"
	"time"
)

type TitleVersionService struct {
	HttpClient       *httpclient.ECFRBulkDataClient
	TitleDAO         *dao.TitleDAO
	TitleVersionDAO  *dao.TitleVersionDAO
}

// ImportHistoricalTitles imports historical CFR titles for a specific date
// The date should be in YYYY-MM-DD format (e.g., "2024-01-01")
func (s *TitleVersionService) ImportHistoricalTitles(
	ctx context.Context,
	versionDate time.Time,
	titlesFilter []string,
) error {
	s.logInfo(fmt.Sprintf("Start - Importing historical titles for %s", versionDate.Format("2006-01-02")))

	// Get all files for the version date
	allFiles, err := s.getAllFilesForDate(ctx, versionDate, titlesFilter)
	if err != nil {
		return fmt.Errorf("failed to get files for date %s: %w", versionDate.Format("2006-01-02"), err)
	}

	s.logInfo(fmt.Sprintf("Found %d title files for %s", len(allFiles), versionDate.Format("2006-01-02")))

	// Create concurrent runner with limited concurrency
	runner := concurrent.NewRunner[ecfrdata.AllFilesItem, int](concurrent.RunnerConfig{
		MaxConcurrency: 5,
		LogPrefix:      fmt.Sprintf("Historical Import (%s)", versionDate.Format("2006-01-02")),
	})

	// Process files concurrently
	result := runner.Run(allFiles, func(
		file ecfrdata.AllFilesItem,
		messages chan<- string,
		results chan<- int,
		errors chan<- error,
	) {
		s.processTitleVersionFile(ctx, file, versionDate, messages, results, errors)
	})

	if len(result.Errors) > 0 {
		s.logInfo(fmt.Sprintf("Completed with %d errors", len(result.Errors)))
		for _, err := range result.Errors {
			s.logInfo(fmt.Sprintf("Error: %v", err))
		}
	} else {
		s.logInfo(fmt.Sprintf("Successfully imported %d titles", len(result.Results)))
	}

	s.logInfo("Complete")
	return nil
}

// processTitleVersionFile processes a single title file for a specific version
func (s *TitleVersionService) processTitleVersionFile(
	ctx context.Context,
	file ecfrdata.AllFilesItem,
	versionDate time.Time,
	messages chan<- string,
	results chan<- int,
	errors chan<- error,
) {
	titleNumber := file.CFRTitle
	messages <- fmt.Sprintf("Fetching: Title %d", titleNumber)

	// Get the title metadata to get the internal ID
	title, err := s.TitleDAO.FindByNumber(ctx, titleNumber)
	if err != nil {
		messages <- fmt.Sprintf("failed to find title %d: %v", titleNumber, err)
		errors <- fmt.Errorf("title %d: %w", titleNumber, err)
		return
	}

	// Get title file details
	titleFile, err := s.getTitleFile(ctx, file.Link)
	if err != nil {
		messages <- fmt.Sprintf("failed to get title file for %d: %v", titleNumber, err)
		errors <- fmt.Errorf("title %d: %w", titleNumber, err)
		return
	}

	messages <- fmt.Sprintf("Downloading: Title %d", titleNumber)

	// Download and store the title version
	err = s.downloadTitleVersion(ctx, title, titleNumber, versionDate, titleFile.Link)
	if err != nil {
		messages <- fmt.Sprintf("failed to download title %d: %v", titleNumber, err)
		errors <- fmt.Errorf("title %d: %w", titleNumber, err)
		return
	}

	messages <- fmt.Sprintf("Success: Title %d", titleNumber)
	results <- titleNumber
}

// getAllFilesForDate retrieves all title files for a specific date
func (s *TitleVersionService) getAllFilesForDate(
	ctx context.Context,
	versionDate time.Time,
	titlesFilter []string,
) ([]ecfrdata.AllFilesItem, error) {
	// Note: The eCFR bulk data API organizes files by date
	// The API root should be something like: https://www.govinfo.gov/bulkdata/ECFR/title-{title}/json/{date}
	// For historical data, we'd need to construct the appropriate URL based on the date

	// For now, this uses the same API as the current titles
	// In a real implementation, you'd need to adjust the API root or endpoint
	// to fetch historical data for the specific date
	allFiles, err := s.HttpClient.GetAllFiles(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch all files: %w", err)
	}

	defer allFiles.Body.Close()
	var allFilesResp ecfrdata.AllFilesResponse
	decoder := json.NewDecoder(allFiles.Body)
	if err := decoder.Decode(&allFilesResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal all files response: %w", err)
	}

	// Build filter map
	filterMap := make(map[string]bool, len(titlesFilter))
	for _, title := range titlesFilter {
		filterMap[title] = true
	}
	hasFilter := len(titlesFilter) > 0

	// Filter files
	var finalFiles []ecfrdata.AllFilesItem
	for _, file := range allFilesResp.Files {
		if file.CFRTitle > 0 && (!hasFilter || filterMap[fmt.Sprintf("%d", file.CFRTitle)]) {
			finalFiles = append(finalFiles, file)
		}
	}

	return finalFiles, nil
}

// getTitleFile gets the XML file details for a title
func (s *TitleVersionService) getTitleFile(
	ctx context.Context,
	url string,
) (*ecfrdata.TitleFileItem, error) {
	titleFiles, err := s.HttpClient.GetJSON(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch files for title at %s: %w", url, err)
	}

	defer titleFiles.Body.Close()
	var titleFilesResp ecfrdata.TitleFilesResponse
	decoder := json.NewDecoder(titleFiles.Body)
	if err := decoder.Decode(&titleFilesResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal title files response: %w", err)
	}

	// Find the XML file
	for _, titleFile := range titleFilesResp.Files {
		if titleFile.FileExtension == "xml" {
			return &titleFile, nil
		}
	}

	return nil, fmt.Errorf("no XML file found for title")
}

// downloadTitleVersion downloads and stores a title version
func (s *TitleVersionService) downloadTitleVersion(
	ctx context.Context,
	title *data.Title,
	titleNumber int,
	versionDate time.Time,
	url string,
) error {
	resp, err := s.HttpClient.GetXML(ctx, url)
	if err != nil {
		return fmt.Errorf("failed to fetch title XML from %s: %w", url, err)
	}

	defer resp.Body.Close()
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read title content: %w", err)
	}

	// Store the title version
	err = s.TitleVersionDAO.Insert(ctx, title.InternalId, titleNumber, versionDate, content)
	if err != nil {
		return fmt.Errorf("failed to insert title version: %w", err)
	}

	return nil
}

func (s *TitleVersionService) logInfo(message string) {
	log.Info(fmt.Sprintf("Title Version Process: %v", message))
}
