ALTER TABLE IF EXISTS ONLY "sources" ADD COLUMN IF NOT EXISTS "path" TEXT;
ALTER TABLE IF EXISTS ONLY "sources" ADD COLUMN IF NOT EXISTS "methods" TEXT[];
ALTER TABLE IF EXISTS ONLY "sources" ADD COLUMN IF NOT EXISTS "response" JSONB;

UPDATE sources SET
  "path" = (config->'http'->>'path'),
  "methods" = ARRAY(SELECT jsonb_array_elements_text(config->'http'->'methods')),
  "response" = (config->'http'->'response');

ALTER TABLE IF EXISTS ONLY "sources" DROP COLUMN IF EXISTS "type";
ALTER TABLE IF EXISTS ONLY "sources" DROP COLUMN IF EXISTS "config";
