-- Reverse of 000002_notifications_seen.

BEGIN;

ALTER TABLE users
    DROP COLUMN IF EXISTS last_notifications_seen_at;

COMMIT;
