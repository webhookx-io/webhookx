CREATE TABLE IF NOT EXISTS "endpoints" (
    "id"          UUID PRIMARY KEY,
    "name"        TEXT UNIQUE,
    "description" TEXT,
    "request"     JSONB NOT NULL DEFAULT '{}'::jsonb,
    "enabled"     BOOLEAN NOT NULL DEFAULT true,
    "metadata"    JSONB NOT NULL DEFAULT '{}'::jsonb,
    "events"      TEXT[],
    "retry"       JSONB NOT NULL DEFAULT '{}'::jsonb,
    "created_at"  TIMESTAMPTZ DEFAULT (CURRENT_TIMESTAMP(0) AT TIME ZONE 'UTC'),
    "updated_at"  TIMESTAMPTZ DEFAULT (CURRENT_TIMESTAMP(0) AT TIME ZONE 'UTC')
);
