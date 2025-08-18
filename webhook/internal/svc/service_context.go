package svc

import (
	"github.com/jackc/pgx/v5"
	"github.com/petechu/idempotent-webhook-relay/db"
	"github.com/petechu/idempotent-webhook-relay/webhook/internal/config"
)

type ServiceContext struct {
	Config   *config.Config
	OutboxDB *db.Queries
}

func NewServiceContext(cfg *config.Config, conn *pgx.Conn) *ServiceContext {
	return &ServiceContext{
		Config:   cfg,
		OutboxDB: db.New(conn),
	}
}
