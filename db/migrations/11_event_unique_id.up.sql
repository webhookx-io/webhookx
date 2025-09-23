ALTER TABLE IF EXISTS ONLY "events" ADD COLUMN IF NOT EXISTS "unique_id" varchar(50);
CREATE UNIQUE INDEX IF NOT EXISTS uk_events_unique_id ON events(unique_id);
