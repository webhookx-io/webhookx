CREATE TABLE IF NOT EXISTS "plugins" (
    "id"          CHAR(27) PRIMARY KEY,
    "endpoint_id" CHAR(27) REFERENCES "endpoints" ("id") ON DELETE CASCADE,
    "name"        TEXT,
    "enabled"     BOOLEAN NOT NULL DEFAULT TRUE,
    "config"      JSONB NOT NULL DEFAULT '{}'::jsonb,
    "ws_id"       CHAR(27),
    "created_at"  TIMESTAMPTZ(3)      DEFAULT (CURRENT_TIMESTAMP(3) AT TIME ZONE 'UTC'),
    "updated_at"  TIMESTAMPTZ(3)      DEFAULT (CURRENT_TIMESTAMP(3) AT TIME ZONE 'UTC')
);

CREATE INDEX idx_plugins_ws_id ON plugins (ws_id);
CREATE UNIQUE INDEX uk_plugins_ws_name ON plugins(endpoint_id, name);
