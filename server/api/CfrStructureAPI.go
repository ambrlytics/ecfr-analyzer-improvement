package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sam-berry/ecfr-analyzer/server/httpresponse"
	"github.com/sam-berry/ecfr-analyzer/server/service"
	"strings"
)

type CfrStructureAPI struct {
	Router              fiber.Router
	CfrStructureService *service.CfrStructureService
}

func (api *CfrStructureAPI) Register() {
	// Admin endpoint to parse and store CFR structure for all titles
	api.Router.Post(
		"/parse/cfr-structure", func(c *fiber.Ctx) error {
			ctx := c.UserContext()
			titles := c.Query("titles")
			var titlesFilter []string
			if len(titles) > 0 {
				titlesFilter = strings.Split(titles, ",")
			} else {
				titlesFilter = []string{}
			}

			err := api.CfrStructureService.ProcessAllTitles(ctx, titlesFilter)

			if err != nil {
				return httpresponse.ApplyErrorToResponse(c, "Unexpected error", err)
			}

			return httpresponse.ApplySuccessToResponse(c, nil)
		},
	)
}
