ALTER TABLE IF EXISTS ONLY "sources" DROP COLUMN IF EXISTS "async";
ALTER TABLE IF EXISTS ONLY "events" DROP COLUMN IF EXISTS "ingested_at";
