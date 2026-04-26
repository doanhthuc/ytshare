-- 000002_notifications_seen: track per-user notifications "last seen" timestamp.
--
-- Used to compute the unread badge count on the bell icon. When the user
-- opens the notifications popover the column is bumped to NOW(); the unread
-- count is the number of videos created after this timestamp that were
-- shared by someone else.

BEGIN;

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS last_notifications_seen_at TIMESTAMPTZ;

COMMIT;
