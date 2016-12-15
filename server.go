package main

import (
	"net/http"

	"github.com/byuoitav/authmiddleware"
	"github.com/byuoitav/touchpanel-update-runner/handlers"
	"github.com/byuoitav/touchpanel-update-runner/helpers"
	"github.com/jessemillar/health"
	"github.com/labstack/echo"
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
	router.Use(middleware.CORS())

	router.GET("/health", echo.WrapHandler(http.HandlerFunc(health.Check)))

	// Use the `secure` routing group to require authentication
	secure := router.Group("", echo.WrapMiddleware(authmiddleware.Authenticate))

	// Touchpanels
	secure.GET("/touchpanel", handlers.GetAllTouchpanelStatus)
	secure.GET("/touchpanel/compact", handlers.GetAllTouchpanelStatusConcise)
	secure.GET("/touchpanel/:address", handlers.GetTouchpanelStatus)

	// Validation
	secure.GET("/validate/touchpanel", handlers.GetValidationStatus)

	secure.POST("/touchpanel", multipleTouchpanelUpdatesHandler)
	secure.POST("/touchpanel/:address", touchpanelUpdateHandler)

	// Callback
	secure.POST("/callback/wait", handlers.WaitCallback)
	secure.POST("/callback/ftp", handlers.FTPCallback)
	secure.POST("/validate/touchpanel", handlers.Validate)

	secure.PUT("/touchpanel", multipleTouchpanelUpdatesHandler)
	secure.PUT("/touchpanel/:address", touchpanelUpdateHandler)

	server := http.Server{
		Addr:           port,
		MaxHeaderBytes: 1024 * 10,
	}

	router.StartServer(&server)
}
