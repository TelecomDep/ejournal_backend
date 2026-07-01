-- +goose Up
-- Add TOTP fields to users and attendance_sessions tables
ALTER TABLE users 
    ADD COLUMN IF NOT EXISTS totp_secret VARCHAR(64),
    ADD COLUMN IF NOT EXISTS is_2fa_enabled BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE attendance_sessions 
    ADD COLUMN IF NOT EXISTS totp_secret VARCHAR(64);

-- +goose Down
ALTER TABLE users 
    DROP COLUMN IF EXISTS totp_secret,
    DROP COLUMN IF EXISTS is_2fa_enabled;

ALTER TABLE attendance_sessions 
    DROP COLUMN IF EXISTS totp_secret;

