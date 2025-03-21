package api_server

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"api.lnlink.net/src/pkg/global"
	"api.lnlink.net/src/pkg/models/user"
	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/checkout/session"
	"github.com/stripe/stripe-go/v81/webhook"
)

func WebhookHandler(c *gin.Context) {
	const MaxBodyBytes = int64(65536)
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxBodyBytes)
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.String(http.StatusRequestEntityTooLarge, "Request body too large")
		return
	}

	sigHeader := c.Request.Header.Get("Stripe-Signature")
	endpointSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
	if endpointSecret == "" {
		c.String(http.StatusInternalServerError, "Webhook secret is not configured")
		return
	}

	event, err := webhook.ConstructEvent(payload, sigHeader, endpointSecret)
	if err != nil {
		log.Printf("Error verifying webhook signature: %v\n", err)
		c.String(http.StatusBadRequest, "Webhook signature verification failed")
		return
	}

	switch event.Type {
	case "checkout.session.completed":
		var stripeSession stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &stripeSession); err != nil {
			log.Printf("Error parsing webhook JSON: %v\n", err)
			c.String(http.StatusBadRequest, "Webhook JSON parsing error")
			return
		}

		customerID := stripeSession.Customer.ID
		if customerID == "" {
			log.Println("No customer ID found in the session")
			c.String(http.StatusBadRequest, "No customer ID in session")
			return
		}

		customerUser := user.GetUserByStripeCustomerID(customerID)
		if customerUser == nil {
			log.Printf("No user found for customer ID: %s\n", customerID)
			c.String(http.StatusBadRequest, "No user found for customer ID")
			return
		}

		params := &stripe.CheckoutSessionParams{
			Expand: []*string{
				stripe.String("line_items"),
			},
		}
		s, err := session.Get(stripeSession.ID, params)
		if err != nil {
			log.Printf("Error getting checkout session: %v\n", err)
			c.String(http.StatusInternalServerError, "Error getting checkout session")
			return
		}

		for _, lineItem := range s.LineItems.Data {
			id := lineItem.Price.ID
			if id == global.TOKENS_10_ID {
				customerUser.AddTokens(10)
			} else if id == global.TOKENS_100_ID {
				customerUser.AddTokens(100)
			} else if id == global.TOKENS_1000_ID {
				customerUser.AddTokens(1000)
			}
		}

	default:
		log.Printf("Unhandled event type: %s\n", event.Type)
	}

	// Respond to Stripe that the webhook was received.
	c.String(http.StatusOK, "Received")
}

func RegisterWebhookRoutes(r *gin.Engine) {
	r.POST("/api/webhooks/stripe", WebhookHandler)
}
