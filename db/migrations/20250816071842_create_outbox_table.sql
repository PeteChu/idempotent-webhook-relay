-- +goose Up
-- +goose StatementBegin
CREATE TYPE outbox_status AS ENUM ('pending', 'processed', 'failed');

CREATE TABLE outbox (
    id SERIAL PRIMARY KEY,
    event_id TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL,
    payload JSONB NOT NULL,
    status outbox_status NOT NULL DEFAULT 'pending',
    retry_count INTEGER NOT NULL DEFAULT 0,
    last_error TEXT,
    last_attempt_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS outbox;
DROP TYPE IF EXISTS outbox_status;
-- +goose StatementEnd
