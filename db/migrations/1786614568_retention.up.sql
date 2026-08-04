CREATE INDEX IF NOT EXISTS idx_events_created_at ON events (created_at);
ALTER TABLE attempts DROP CONSTRAINT IF EXISTS attempts_endpoint_id_fkey;
ALTER TABLE attempts DROP CONSTRAINT IF EXISTS  attempts_event_id_fkey;
CREATE INDEX IF NOT EXISTS idx_attempts_created_at ON attempts (created_at);
