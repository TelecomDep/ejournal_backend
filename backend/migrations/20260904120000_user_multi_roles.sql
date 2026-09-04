-- +goose Up
-- +goose StatementBegin
-- A user keeps one primary role in users.role for backwards compatibility,
-- while every effective role is stored in this many-to-many table.
CREATE TABLE user_roles (
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role user_role NOT NULL,
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    assigned_by INT REFERENCES users(id) ON DELETE SET NULL,
    PRIMARY KEY (user_id, role)
);

CREATE UNIQUE INDEX user_roles_one_primary_per_user
    ON user_roles (user_id)
    WHERE is_primary;

CREATE INDEX user_roles_role_user_idx
    ON user_roles (role, user_id);

INSERT INTO user_roles (user_id, role, is_primary)
SELECT id, role, TRUE
FROM users
ON CONFLICT (user_id, role) DO UPDATE SET is_primary = TRUE;

-- Heads already had teaching access in the application. Preserve it as an
-- explicit assignment when a teaching profile is present.
INSERT INTO user_roles (user_id, role, is_primary)
SELECT DISTINCT u.id, 'teacher'::user_role, FALSE
FROM users u
JOIN teachers t ON t.user_id = u.id OR t.teacher_id = u.id
WHERE u.role = 'head'
ON CONFLICT (user_id, role) DO NOTHING;

CREATE OR REPLACE FUNCTION sync_user_primary_role_assignment() RETURNS trigger AS $$
BEGIN
    UPDATE user_roles
    SET is_primary = FALSE
    WHERE user_id = NEW.id AND is_primary;

    INSERT INTO user_roles (user_id, role, is_primary)
    VALUES (NEW.id, NEW.role, TRUE)
    ON CONFLICT (user_id, role) DO UPDATE
        SET is_primary = TRUE;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_sync_user_primary_role_assignment ON users;
CREATE TRIGGER trg_sync_user_primary_role_assignment
AFTER INSERT OR UPDATE OF role ON users
FOR EACH ROW
EXECUTE FUNCTION sync_user_primary_role_assignment();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS trg_sync_user_primary_role_assignment ON users;
DROP FUNCTION IF EXISTS sync_user_primary_role_assignment();
DROP TABLE IF EXISTS user_roles;
-- +goose StatementEnd
