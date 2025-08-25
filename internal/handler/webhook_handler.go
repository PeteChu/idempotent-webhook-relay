package handler

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	_ "github.com/joho/godotenv/autoload"
	"github.com/petechu/idempotent-webhook-relay/internal/logic"
	"github.com/petechu/idempotent-webhook-relay/internal/svc"
	"github.com/stripe/stripe-go/v82/webhook"
)

func stripeWebhookHandler(svcCtx *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		const MaxBodyBytes = int64(65536)
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxBodyBytes)

		payload, err := io.ReadAll(c.Request.Body)
		if err != nil {
			fmt.Println("Error reading request body:", err)
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"message": "Error reading request body",
			})
			return
		}

		signature := c.GetHeader("Stripe-Signature")
		event, err := webhook.ConstructEvent(payload, signature, svcCtx.Config.StripeWebhookSecret)
		if err != nil {
			fmt.Println("Error verifying webhook signature:", err)
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "Invalid signature",
			})
			return
		}

		l := logic.NewStoreStripeEventLogic(c.Request.Context(), svcCtx)
		if err := l.StoreStripeEvent(event); err != nil {
			fmt.Println("Error storing event:", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": fmt.Sprintf("Failed to store event: %s", err),
			})
			return
		}

		c.JSON(200, gin.H{
			"message": "Webhook received",
		})
	}
}
