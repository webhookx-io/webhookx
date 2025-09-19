ALTER TABLE IF EXISTS ONLY "events" ADD COLUMN IF NOT EXISTS "key" varchar(50);
CREATE UNIQUE INDEX uk_events_key ON events(key);
