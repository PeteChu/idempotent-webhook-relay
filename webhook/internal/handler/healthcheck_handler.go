package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/petechu/idempotent-webhook-relay/webhook/internal/svc"
)

func healthCheckHandler(svcCtx *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.String(200, "OK")
	}
}
