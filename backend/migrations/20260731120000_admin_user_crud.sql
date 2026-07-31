-- +goose Up
-- +goose StatementBegin
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'active',
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_status_check;
ALTER TABLE users
    ADD CONSTRAINT users_status_check
    CHECK (status IN ('active', 'blocked', 'archived'));

CREATE INDEX IF NOT EXISTS idx_users_role_status
    ON users (role, status);

CREATE INDEX IF NOT EXISTS idx_users_login_lower
    ON users (LOWER(login));

CREATE OR REPLACE FUNCTION set_users_updated_at() RETURNS trigger AS $$
BEGIN
    NEW.updated_at := NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_users_updated_at ON users;
CREATE TRIGGER trg_users_updated_at
BEFORE UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION set_users_updated_at();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS trg_users_updated_at ON users;
DROP FUNCTION IF EXISTS set_users_updated_at();
DROP INDEX IF EXISTS idx_users_login_lower;
DROP INDEX IF EXISTS idx_users_role_status;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_status_check;
ALTER TABLE users
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS status;
-- +goose StatementEnd
