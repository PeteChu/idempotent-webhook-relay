package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/petechu/idempotent-webhook-relay/webhook/internal/logic"
	"github.com/petechu/idempotent-webhook-relay/webhook/internal/svc"
	"github.com/stripe/stripe-go/v82"
)

func webhookHandler(svcCtx *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		event := stripe.Event{}
		if err := c.ShouldBindBodyWithJSON(&event); err != nil {
			c.JSON(400, gin.H{
				"message": "Invalid request body",
			})
			return
		}

		l := logic.NewStoreStripeEventLogic(c.Request.Context(), svcCtx)
		l.StoreStripeEvent(event)
		c.JSON(200, gin.H{
			"message": "Webhook received",
		})
	}
}
