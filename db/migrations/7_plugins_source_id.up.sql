ALTER TABLE IF EXISTS ONLY "plugins" ADD COLUMN IF NOT EXISTS "source_id" CHAR(27) REFERENCES "sources" ("id") ON DELETE CASCADE;
CREATE UNIQUE INDEX uk_plugins_source_name ON plugins(source_id, name);
