ALTER TABLE IF EXISTS ONLY "attempt_details" ALTER COLUMN request_body TYPE JSONB USING request_body::JSONB;
ALTER TABLE IF EXISTS ONLY "attempt_details" ALTER COLUMN response_body TYPE JSONB USING response_body::JSONB;
