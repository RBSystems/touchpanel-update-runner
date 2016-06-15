package main

import (
	"flag"
	"fmt"

	"github.com/byuoitav/touchpanel-update-runner/controllers"
	"github.com/byuoitav/touchpanel-update-runner/helpers"
	"github.com/jessemillar/health"
	"github.com/labstack/echo"
	"github.com/labstack/echo/engine/standard"
	"github.com/labstack/echo/middleware"
)

func main() {
	flag.Parse()

	// Build our channels
	submissionChannel := make(chan helpers.TouchpanelStatus, 50)         // Only used in server.go
	helpers.UpdateChannel = make(chan helpers.TouchpanelStatus, 150)     // Global
	helpers.ValidationChannel = make(chan helpers.TouchpanelStatus, 150) // Global

	// Populate miscellaneous globals
	helpers.TouchpanelStatusMap = make(map[string]helpers.TouchpanelStatus)
	helpers.ValidationStatus = make(map[string]helpers.TouchpanelStatus)

	// Watch for new things in channels
	go helpers.ValidateHelper()
	go helpers.ChannelUpdater()

	// Build a couple controllers (to have access to channels, controllers must be wrapped)
	touchpanelUpdateController := controllers.BuildControllerStartTouchpanelUpdate(submissionChannel)
	multipleTouchpanelUpdatesController := controllers.BuildControllerStartMultipleTPUpdate(submissionChannel)

	port := ":8004"
	e := echo.New()
	e.Pre(middleware.RemoveTrailingSlash())

	e.Get("/health", health.Check)

	// Touchpanels
	e.Get("/touchpanel", controllers.GetAllTouchpanelStatus)
	e.Get("/touchpanel/compact", controllers.GetAllTouchpanelStatusConcise)
	e.Get("/touchpanel/:address", controllers.GetTouchpanelStatus)

	e.Post("/touchpanel", multipleTouchpanelUpdatesController)
	e.Post("/touchpanel/:address", touchpanelUpdateController)

	e.Put("/touchpanel", multipleTouchpanelUpdatesController)
	e.Put("/touchpanel/:address", touchpanelUpdateController)

	// Callback
	e.Post("/callback/wait", controllers.WaitCallback)
	e.Post("/callback/ftp", controllers.FTPCallback)

	// Validation
	e.Get("/validate/touchpanel", controllers.GetValidationStatus)

	e.Post("/validate/touchpanel", controllers.Validate)

	fmt.Printf("The Touchpanel Update Runner is listening on %s\n", port)
	e.Run(standard.New(port))
}
