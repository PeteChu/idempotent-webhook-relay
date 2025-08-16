# --- Config ---
dsn ?= "host=localhost port=5432 user=postgres password=postgres dbname=idempotent-webhook-relay sslmode=disable"
migrationPath ?= "./db/migrations"


db-status:
	@GOOSE_DRIVER=postgres GOOSE_DBSTRING=$(dsn) goose -dir=$(migrationPath) status

up:
	@GOOSE_DRIVER=postgres GOOSE_DBSTRING=$(dsn) goose -dir=$(migrationPath) up


reset:
	@GOOSE_DRIVER=postgres GOOSE_DBSTRING=$(dsn) goose -dir=$(migrationPath) reset
