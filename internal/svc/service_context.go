package svc

import (
	"github.com/jackc/pgx/v5"
	"github.com/petechu/idempotent-webhook-relay/internal/config"
	"github.com/petechu/idempotent-webhook-relay/internal/db"
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
