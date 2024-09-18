CREATE TABLE IF NOT EXISTS "workspaces" (
    "id"          CHAR(27) PRIMARY KEY,
    "name"        TEXT UNIQUE,
    "description" TEXT,

    "created_at"  TIMESTAMPTZ(3)    DEFAULT (CURRENT_TIMESTAMP(3) AT TIME ZONE 'UTC'),
    "updated_at"  TIMESTAMPTZ(3)    DEFAULT (CURRENT_TIMESTAMP(3) AT TIME ZONE 'UTC')
);

CREATE TABLE IF NOT EXISTS "endpoints" (
    "id"          CHAR(27) PRIMARY KEY,
    "name"        TEXT,
    "description" TEXT,
    "request"     JSONB   NOT NULL DEFAULT '{}'::jsonb,
    "enabled"     BOOLEAN NOT NULL DEFAULT true,
    "metadata"    JSONB   NOT NULL DEFAULT '{}'::jsonb,
    "events"      TEXT[],
    "retry"       JSONB   NOT NULL DEFAULT '{}'::jsonb,

    "ws_id"       CHAR(27),
    "created_at"  TIMESTAMPTZ(3)      DEFAULT (CURRENT_TIMESTAMP(3) AT TIME ZONE 'UTC'),
    "updated_at"  TIMESTAMPTZ(3)      DEFAULT (CURRENT_TIMESTAMP(3) AT TIME ZONE 'UTC')
);

CREATE INDEX idx_endpoints_ws_id ON endpoints (ws_id);
CREATE UNIQUE INDEX uk_endpoints_ws_name ON endpoints (ws_id, name);

CREATE TABLE IF NOT EXISTS "events" (
    "id"         CHAR(27) PRIMARY KEY,
    "data"       JSONB NOT NULL,
    "event_type" TEXT  NOT NULL,

    "ws_id"      CHAR(27),
    "created_at" TIMESTAMPTZ(3) DEFAULT (CURRENT_TIMESTAMP(3) AT TIME ZONE 'UTC'),
    "updated_at" TIMESTAMPTZ(3) DEFAULT (CURRENT_TIMESTAMP(3) AT TIME ZONE 'UTC')
);

CREATE INDEX idx_events_ws_id ON events (ws_id);

CREATE TABLE IF NOT EXISTS "attempts" (
    "id"             CHAR(27) PRIMARY KEY,
    "event_id"       CHAR(27) REFERENCES "events" ("id") ON DELETE CASCADE,
    "endpoint_id"    CHAR(27) REFERENCES "endpoints" ("id") ON DELETE CASCADE,
    "status"         VARCHAR(20) not null,

    "attempt_number" SMALLINT    NOT NULL DEFAULT 1,
    "scheduled_at"   TIMESTAMPTZ(3),
    "attempted_at"   TIMESTAMPTZ(3),
    "error_code"     VARCHAR(30),

    "request"        JSONB,
    "response"       JSONB,

    "ws_id"          CHAR(27),
    "created_at"     TIMESTAMPTZ(3)          DEFAULT (CURRENT_TIMESTAMP(3) AT TIME ZONE 'UTC'),
    "updated_at"     TIMESTAMPTZ(3)          DEFAULT (CURRENT_TIMESTAMP(3) AT TIME ZONE 'UTC')
);

CREATE INDEX idx_attempts_event_id ON attempts (event_id);
CREATE INDEX idx_attempts_endpoint_id ON attempts (endpoint_id);
CREATE INDEX idx_attempts_ws_id ON attempts (ws_id);
CREATE INDEX idx_attempts_status ON attempts (status);

CREATE TABLE IF NOT EXISTS "sources" (
    "id"          CHAR(27) PRIMARY KEY,
    "name"        TEXT UNIQUE,
    "enabled"     BOOLEAN NOT NULL DEFAULT true,

    "path"        TEXT,
    "methods"     TEXT[],
    "response"    JSONB,

    "ws_id"       CHAR(27),
    "created_at"  TIMESTAMPTZ(3) DEFAULT (CURRENT_TIMESTAMP(3) AT TIME ZONE 'UTC'),
    "updated_at"  TIMESTAMPTZ(3) DEFAULT (CURRENT_TIMESTAMP(3) AT TIME ZONE 'UTC')
);

CREATE INDEX idx_sources_ws_id ON sources (ws_id);
CREATE UNIQUE INDEX uk_sources_ws_name ON sources (ws_id, name);
