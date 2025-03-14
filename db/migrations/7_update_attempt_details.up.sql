ALTER TABLE IF EXISTS ONLY "attempt_details" ALTER COLUMN request_headers TYPE TEXT;
ALTER TABLE IF EXISTS ONLY "attempt_details" ALTER COLUMN response_headers TYPE TEXT;
DROP INDEX idx_attempt_details_ws_id;
