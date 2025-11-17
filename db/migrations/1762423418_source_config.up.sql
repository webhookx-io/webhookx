ALTER TABLE IF EXISTS ONLY "sources" ADD COLUMN IF NOT EXISTS "type" varchar(20);
ALTER TABLE IF EXISTS ONLY "sources" ADD COLUMN IF NOT EXISTS "config" JSONB NOT NULL DEFAULT '{}'::jsonb;

UPDATE sources SET "type" = 'http', "config" = jsonb_build_object('http', jsonb_build_object('methods', methods, 'path', path, 'response', response));

ALTER TABLE IF EXISTS ONLY "sources" DROP COLUMN IF EXISTS "path";
ALTER TABLE IF EXISTS ONLY "sources" DROP COLUMN IF EXISTS "methods";
ALTER TABLE IF EXISTS ONLY "sources" DROP COLUMN IF EXISTS "response";
