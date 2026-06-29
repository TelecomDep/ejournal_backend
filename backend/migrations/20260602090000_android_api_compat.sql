-- +goose Up
-- +goose StatementBegin
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS avatar_url TEXT;

ALTER TABLE students
    ADD COLUMN IF NOT EXISTS total_cheat_attempts INTEGER NOT NULL DEFAULT 0;

ALTER TABLE attendance_sessions
    ADD COLUMN IF NOT EXISTS lesson_name VARCHAR(255);

ALTER TABLE attendance_sessions
    ADD COLUMN IF NOT EXISTS lat DOUBLE PRECISION;

ALTER TABLE attendance_sessions
    ADD COLUMN IF NOT EXISTS lon DOUBLE PRECISION;

ALTER TABLE attendance_session_students
    ADD COLUMN IF NOT EXISTS device_id VARCHAR(255);

ALTER TABLE attendance_session_students
    ADD COLUMN IF NOT EXISTS check_in_lat DOUBLE PRECISION;

ALTER TABLE attendance_session_students
    ADD COLUMN IF NOT EXISTS check_in_lon DOUBLE PRECISION;

ALTER TABLE attendance_session_students
    ADD COLUMN IF NOT EXISTS is_fraud BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE attendance_session_students
    ADD COLUMN IF NOT EXISTS fraud_reason TEXT;

CREATE INDEX IF NOT EXISTS idx_attendance_session_students_session_device
    ON attendance_session_students (session_id, device_id)
    WHERE device_id IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_attendance_session_students_session_device;

ALTER TABLE attendance_session_students DROP COLUMN IF EXISTS fraud_reason;
ALTER TABLE attendance_session_students DROP COLUMN IF EXISTS is_fraud;
ALTER TABLE attendance_session_students DROP COLUMN IF EXISTS check_in_lon;
ALTER TABLE attendance_session_students DROP COLUMN IF EXISTS check_in_lat;
ALTER TABLE attendance_session_students DROP COLUMN IF EXISTS device_id;

ALTER TABLE attendance_sessions DROP COLUMN IF EXISTS lon;
ALTER TABLE attendance_sessions DROP COLUMN IF EXISTS lat;
ALTER TABLE attendance_sessions DROP COLUMN IF EXISTS lesson_name;

ALTER TABLE students DROP COLUMN IF EXISTS total_cheat_attempts;
ALTER TABLE users DROP COLUMN IF EXISTS avatar_url;
-- +goose StatementEnd
