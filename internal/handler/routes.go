package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/petechu/idempotent-webhook-relay/internal/svc"
)

func RegisterRoutes(r *gin.Engine, svcCtx *svc.ServiceContext) {
	r.Use(CORS())

	r.GET("/healthz", healthCheckHandler(svcCtx))
	r.POST("/stripe/webhook", stripeWebhookHandler(svcCtx))
}
