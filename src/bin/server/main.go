package main

import (
	"api.lnlink.net/src/pkg/api_server"
	"api.lnlink.net/src/pkg/global"

	"github.com/gin-contrib/cors"
)

func main() {
	// Connect to Mongo
	global.Init()
	defer global.Deinit()

	// Configure CORS
	global.GIN_ROUTER.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "PATCH", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	api_server.RegisterAllRoutes(global.GIN_ROUTER)
	err := global.GIN_ROUTER.Run()
	if err != nil {
		return
	}
}
