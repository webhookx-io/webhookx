ALTER TABLE IF EXISTS ONLY "attempt_details" ALTER COLUMN request_headers TYPE JSONB USING request_headers::JSONB;
ALTER TABLE IF EXISTS ONLY "attempt_details" ALTER COLUMN response_headers TYPE JSONB USING response_headers::JSONB;
CREATE INDEX idx_attempt_details_ws_id ON attempt_details (ws_id);
