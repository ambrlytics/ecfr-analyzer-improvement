package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sam-berry/ecfr-analyzer/server/httpresponse"
	"github.com/sam-berry/ecfr-analyzer/server/service"
	"strings"
	"time"
)

type ChangeTrackingAPI struct {
	Router                fiber.Router
	ChangeTrackingService *service.ChangeTrackingService
}

func (api *ChangeTrackingAPI) Register() {
	// Admin endpoint to compute changes between two dates
	api.Router.Post(
		"/compute/changes", func(c *fiber.Ctx) error {
			ctx := c.UserContext()

			// Get date parameters (required)
			startDateStr := c.Query("startDate") // Format: YYYY-MM-DD
			endDateStr := c.Query("endDate")     // Format: YYYY-MM-DD

			if startDateStr == "" || endDateStr == "" {
				return httpresponse.ApplyErrorToResponse(c, "startDate and endDate parameters are required (format: YYYY-MM-DD)", nil)
			}

			startDate, err := time.Parse("2006-01-02", startDateStr)
			if err != nil {
				return httpresponse.ApplyErrorToResponse(c, "Invalid startDate format. Use YYYY-MM-DD", err)
			}

			endDate, err := time.Parse("2006-01-02", endDateStr)
			if err != nil {
				return httpresponse.ApplyErrorToResponse(c, "Invalid endDate format. Use YYYY-MM-DD", err)
			}

			// Get optional titles filter
			titles := c.Query("titles")
			var titlesFilter []string
			if len(titles) > 0 {
				titlesFilter = strings.Split(titles, ",")
			} else {
				titlesFilter = []string{}
			}

			err = api.ChangeTrackingService.ComputeChangesForDateRange(ctx, startDate, endDate, titlesFilter)

			if err != nil {
				return httpresponse.ApplyErrorToResponse(c, "Unexpected error", err)
			}

			return httpresponse.ApplySuccessToResponse(c, nil)
		},
	)

	// Public endpoint to get change summary
	api.Router.Get(
		"/changes/summary", func(c *fiber.Ctx) error {
			ctx := c.UserContext()

			// Get date parameters (required)
			startDateStr := c.Query("startDate") // Format: YYYY-MM-DD
			endDateStr := c.Query("endDate")     // Format: YYYY-MM-DD

			if startDateStr == "" || endDateStr == "" {
				return httpresponse.ApplyErrorToResponse(c, "startDate and endDate parameters are required (format: YYYY-MM-DD)", nil)
			}

			startDate, err := time.Parse("2006-01-02", startDateStr)
			if err != nil {
				return httpresponse.ApplyErrorToResponse(c, "Invalid startDate format. Use YYYY-MM-DD", err)
			}

			endDate, err := time.Parse("2006-01-02", endDateStr)
			if err != nil {
				return httpresponse.ApplyErrorToResponse(c, "Invalid endDate format. Use YYYY-MM-DD", err)
			}

			changes, err := api.ChangeTrackingService.GetChangeSummary(ctx, startDate, endDate)
			if err != nil {
				return httpresponse.ApplyErrorToResponse(c, "Unexpected error", err)
			}

			return httpresponse.ApplySuccessToResponse(c, changes)
		},
	)

	// Public endpoint to get top changing titles
	api.Router.Get(
		"/changes/top", func(c *fiber.Ctx) error {
			ctx := c.UserContext()

			// Get date parameters (required)
			startDateStr := c.Query("startDate") // Format: YYYY-MM-DD
			endDateStr := c.Query("endDate")     // Format: YYYY-MM-DD

			if startDateStr == "" || endDateStr == "" {
				return httpresponse.ApplyErrorToResponse(c, "startDate and endDate parameters are required (format: YYYY-MM-DD)", nil)
			}

			startDate, err := time.Parse("2006-01-02", startDateStr)
			if err != nil {
				return httpresponse.ApplyErrorToResponse(c, "Invalid startDate format. Use YYYY-MM-DD", err)
			}

			endDate, err := time.Parse("2006-01-02", endDateStr)
			if err != nil {
				return httpresponse.ApplyErrorToResponse(c, "Invalid endDate format. Use YYYY-MM-DD", err)
			}

			// Get optional limit parameter (default: 10)
			limit := c.QueryInt("limit", 10)

			topChanges, err := api.ChangeTrackingService.GetTopChangingTitles(ctx, startDate, endDate, limit)
			if err != nil {
				return httpresponse.ApplyErrorToResponse(c, "Unexpected error", err)
			}

			return httpresponse.ApplySuccessToResponse(c, topChanges)
		},
	)

	// Public endpoint to generate a change report
	api.Router.Get(
		"/changes/report", func(c *fiber.Ctx) error {
			ctx := c.UserContext()

			// Get date parameters (required)
			startDateStr := c.Query("startDate") // Format: YYYY-MM-DD
			endDateStr := c.Query("endDate")     // Format: YYYY-MM-DD

			if startDateStr == "" || endDateStr == "" {
				return httpresponse.ApplyErrorToResponse(c, "startDate and endDate parameters are required (format: YYYY-MM-DD)", nil)
			}

			startDate, err := time.Parse("2006-01-02", startDateStr)
			if err != nil {
				return httpresponse.ApplyErrorToResponse(c, "Invalid startDate format. Use YYYY-MM-DD", err)
			}

			endDate, err := time.Parse("2006-01-02", endDateStr)
			if err != nil {
				return httpresponse.ApplyErrorToResponse(c, "Invalid endDate format. Use YYYY-MM-DD", err)
			}

			report, err := api.ChangeTrackingService.GenerateChangeReport(ctx, startDate, endDate)
			if err != nil {
				return httpresponse.ApplyErrorToResponse(c, "Unexpected error", err)
			}

			// Return as plain text
			c.Set("Content-Type", "text/plain")
			return c.SendString(report)
		},
	)
}
