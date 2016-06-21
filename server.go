package main

import (
	"log"

	"github.com/byuoitav/touchpanel-update-runner/handlers"
	"github.com/byuoitav/touchpanel-update-runner/helpers"
	"github.com/jessemillar/health"
	"github.com/labstack/echo"
	"github.com/labstack/echo/engine/fasthttp"
	"github.com/labstack/echo/middleware"
)

func main() {
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

	// Build a couple handlers (to have access to channels, handlers must be wrapped)
	touchpanelUpdateHandler := handlers.BuildHandlerStartTouchpanelUpdate(submissionChannel)
	multipleTouchpanelUpdatesHandler := handlers.BuildHandlerStartMultipleTPUpdate(submissionChannel)

	port := ":8004"
	router := echo.New()
	router.Pre(middleware.RemoveTrailingSlash())

	router.Get("/health", health.Check)

	// Touchpanels
	router.Get("/touchpanel", handlers.GetAllTouchpanelStatus)
	router.Get("/touchpanel/compact", handlers.GetAllTouchpanelStatusConcise)
	router.Get("/touchpanel/:address", handlers.GetTouchpanelStatus)

	router.Post("/touchpanel", multipleTouchpanelUpdatesHandler)
	router.Post("/touchpanel/:address", touchpanelUpdateHandler)

	router.Put("/touchpanel", multipleTouchpanelUpdatesHandler)
	router.Put("/touchpanel/:address", touchpanelUpdateHandler)

	// Callback
	router.Post("/callback/wait", handlers.WaitCallback)
	router.Post("/callback/ftp", handlers.FTPCallback)

	// Validation
	router.Get("/validate/touchpanel", handlers.GetValidationStatus)

	router.Post("/validate/touchpanel", handlers.Validate)

	log.Println("The Touchpanel Update Runner is listening on " + port)
	server := fasthttp.New(port)
	server.ReadBufferSize = 1024 * 10 // Needed to interface properly with WSO2
	router.Run(server)
}
