package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sam-berry/ecfr-analyzer/server/httpresponse"
	"github.com/sam-berry/ecfr-analyzer/server/service"
	"strings"
	"time"
)

type TitleVersionAPI struct {
	Router              fiber.Router
	TitleVersionService *service.TitleVersionService
}

func (api *TitleVersionAPI) Register() {
	// Admin endpoint to import historical CFR titles for a specific date
	api.Router.Post(
		"/import/historical-titles", func(c *fiber.Ctx) error {
			ctx := c.UserContext()

			// Get date parameter (required)
			dateStr := c.Query("date") // Format: YYYY-MM-DD
			if dateStr == "" {
				return httpresponse.ApplyErrorToResponse(c, "Date parameter is required (format: YYYY-MM-DD)", nil)
			}

			versionDate, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				return httpresponse.ApplyErrorToResponse(c, "Invalid date format. Use YYYY-MM-DD", err)
			}

			// Get optional titles filter
			titles := c.Query("titles")
			var titlesFilter []string
			if len(titles) > 0 {
				titlesFilter = strings.Split(titles, ",")
			} else {
				titlesFilter = []string{}
			}

			err = api.TitleVersionService.ImportHistoricalTitles(ctx, versionDate, titlesFilter)

			if err != nil {
				return httpresponse.ApplyErrorToResponse(c, "Unexpected error", err)
			}

			return httpresponse.ApplySuccessToResponse(c, nil)
		},
	)
}
