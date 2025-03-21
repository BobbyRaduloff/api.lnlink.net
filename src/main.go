package main

import (
	"api.lnlink.net/src/pkg/api_server"
	"api.lnlink.net/src/pkg/global"
	"api.lnlink.net/src/pkg/services/cron"
)

func main() {
	global.Init()
	defer global.Deinit()

	// Start the experiment status cron job
	cron.StartExperimentStatusCron()

	// Register all routes
	api_server.RegisterAllRoutes(global.GIN_ROUTER)

	// Start the server
	global.GIN_ROUTER.Run(":8080")
}
