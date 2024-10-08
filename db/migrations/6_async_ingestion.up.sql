ALTER TABLE IF EXISTS ONLY "sources" ADD COLUMN IF NOT EXISTS "async" BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE IF EXISTS ONLY "events" ADD COLUMN IF NOT EXISTS "ingested_at" TIMESTAMPTZ(3) DEFAULT (CURRENT_TIMESTAMP(3) AT TIME ZONE 'UTC');

UPDATE "events" SET ingested_at = created_at;
