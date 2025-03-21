package api_server

import (
	"github.com/gin-gonic/gin"
)

func RegisterAllRoutes(r *gin.Engine) {
	RegisterAuthRoutes(r)
	RegisterPurchasingRoutes(r)
	RegisterWebhookRoutes(r)
}
