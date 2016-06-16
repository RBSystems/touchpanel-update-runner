package main

import (
	"flag"
	"fmt"

	"github.com/byuoitav/touchpanel-update-runner/controllers"
	"github.com/byuoitav/touchpanel-update-runner/helpers"
	"github.com/jessemillar/health"
	"github.com/labstack/echo"
	"github.com/labstack/echo/engine/fasthttp"
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
	router := echo.New()
	router.Pre(middleware.RemoveTrailingSlash())

	router.Get("/health", health.Check)

	// Touchpanels
	router.Get("/touchpanel", controllers.GetAllTouchpanelStatus)
	router.Get("/touchpanel/compact", controllers.GetAllTouchpanelStatusConcise)
	router.Get("/touchpanel/:address", controllers.GetTouchpanelStatus)

	router.Post("/touchpanel", multipleTouchpanelUpdatesController)
	router.Post("/touchpanel/:address", touchpanelUpdateController)

	router.Put("/touchpanel", multipleTouchpanelUpdatesController)
	router.Put("/touchpanel/:address", touchpanelUpdateController)

	// Callback
	router.Post("/callback/wait", controllers.WaitCallback)
	router.Post("/callback/ftp", controllers.FTPCallback)

	// Validation
	router.Get("/validate/touchpanel", controllers.GetValidationStatus)

	router.Post("/validate/touchpanel", controllers.Validate)

	fmt.Printf("The Touchpanel Update Runner is listening on %s\n", port)
	server := fasthttp.New(port)
	server.ReadBufferSize = 1024 * 10 // Needed to interface properly with WSO2
	router.Run(server)
}
