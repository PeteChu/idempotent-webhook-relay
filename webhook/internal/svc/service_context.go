package svc

import (
	"github.com/jackc/pgx/v5"
	"github.com/petechu/idempotent-webhook-relay/db"
)

type ServiceContext struct {
	OutboxDB *db.Queries
}

func NewServiceContext(conn *pgx.Conn) *ServiceContext {
	return &ServiceContext{
		OutboxDB: db.New(conn),
	}
}
