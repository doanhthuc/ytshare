-- 000001_init: bootstrap schema for users and shared videos.
--
-- This is the first versioned migration. Subsequent changes go in
-- new files (000002_xxx, 000003_xxx, …); never edit applied files
-- — that's the whole point of Flyway-style migrations.

BEGIN;

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS users (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    email           VARCHAR(320) NOT NULL,
    name            VARCHAR(120) NOT NULL,
    password_hash   VARCHAR(255) NOT NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users (LOWER(email));

CREATE TABLE IF NOT EXISTS videos (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    youtube_id      VARCHAR(32)  NOT NULL,
    url             VARCHAR(512) NOT NULL,
    title           VARCHAR(255) NOT NULL,
    description     TEXT         NOT NULL DEFAULT '',
    thumbnail_url   VARCHAR(512) NOT NULL DEFAULT '',
    shared_by_id    UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_videos_youtube_id ON videos (youtube_id);
CREATE INDEX IF NOT EXISTS idx_videos_shared_by_id ON videos (shared_by_id);
CREATE INDEX IF NOT EXISTS idx_videos_created_at ON videos (created_at DESC);

COMMIT;
