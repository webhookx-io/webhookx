ALTER TABLE IF EXISTS ONLY "attempts" ADD COLUMN IF NOT EXISTS "trigger_mode" VARCHAR(10) NOT NULL;
ALTER TABLE IF EXISTS ONLY "attempts" ADD COLUMN IF NOT EXISTS "exhausted" BOOLEAN NOT NULL DEFAULT false;

UPDATE "attempts" SET trigger_mode = 'INITIAL' where attempt_number = 1;
UPDATE "attempts" SET trigger_mode = 'AUTOMATIC' where attempt_number != 1;
