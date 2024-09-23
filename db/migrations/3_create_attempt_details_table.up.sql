CREATE TABLE IF NOT EXISTS "attempt_details" (
    "id"             CHAR(27) PRIMARY KEY REFERENCES "attempts" ("id") ON DELETE CASCADE,

    "request_headers"     JSONB,
    "request_body"        JSONB,
    "response_headers"    JSONB,
    "response_body"       JSONB,

    "ws_id"          CHAR(27),
    "created_at"     TIMESTAMPTZ(3)          DEFAULT (CURRENT_TIMESTAMP(3) AT TIME ZONE 'UTC'),
    "updated_at"     TIMESTAMPTZ(3)          DEFAULT (CURRENT_TIMESTAMP(3) AT TIME ZONE 'UTC')
);

CREATE INDEX idx_attempt_details_ws_id ON attempt_details (ws_id);
