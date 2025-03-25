package api_server

import (
	"net/http"

	"api.lnlink.net/src/pkg/global"
	"api.lnlink.net/src/pkg/models/user"
	"api.lnlink.net/src/pkg/services/stripe"
	"github.com/gin-gonic/gin"
)

func CreateCheckoutSession(c *gin.Context) {
	userID := GetUserID(c)
	user := user.GetUserByID(userID)
	tokens := c.Param("tokens")

	var checkoutSession string
	var err error
	switch tokens {
	case "5000":
		checkoutSession, err = stripe.CreateCheckoutSession(user.StripeCustomerID, global.TOKENS_5000_ID)
	case "100":
		checkoutSession, err = stripe.CreateCheckoutSession(user.StripeCustomerID, global.TOKENS_100_ID)
	case "1000":
		checkoutSession, err = stripe.CreateCheckoutSession(user.StripeCustomerID, global.TOKENS_1000_ID)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid number of tokens"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"checkoutSession": checkoutSession})
}

func RegisterPurchasingRoutes(r *gin.Engine) {
	r.GET("/api/purchasing/checkout/:tokens", AuthMiddleware(), CreateCheckoutSession)
}
