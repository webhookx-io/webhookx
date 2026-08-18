DROP INDEX IF EXISTS idx_events_created_at;
DROP INDEX IF EXISTS idx_attempts_created_at;
ALTER TABLE attempts ADD CONSTRAINT attempts_event_id_fkey FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE;
ALTER TABLE attempts ADD CONSTRAINT attempts_endpoint_id_fkey FOREIGN KEY (endpoint_id) REFERENCES endpoints(id) ON DELETE CASCADE;
