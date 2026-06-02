-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS registration_invites (
    invite_id SERIAL PRIMARY KEY,
    invite_code VARCHAR(100) NOT NULL,
    invite_code_hash TEXT NOT NULL,
    role user_role NOT NULL,
    student_id INTEGER REFERENCES students(student_id) ON DELETE CASCADE,
    teacher_id INTEGER REFERENCES teachers(teacher_id) ON DELETE CASCADE,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT registration_invites_role_target_check CHECK (
        (
            role = 'student'::user_role
            AND student_id IS NOT NULL
            AND teacher_id IS NULL
        ) OR (
            role = 'teacher'::user_role
            AND teacher_id IS NOT NULL
            AND student_id IS NULL
        ) OR (
            role = 'admin'::user_role
            AND student_id IS NULL
            AND teacher_id IS NULL
        )
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_registration_invites_invite_code_unique
    ON registration_invites (invite_code);

CREATE UNIQUE INDEX IF NOT EXISTS idx_registration_invites_invite_code_hash_unique
    ON registration_invites (invite_code_hash);

CREATE INDEX IF NOT EXISTS idx_registration_invites_role
    ON registration_invites (role);

CREATE INDEX IF NOT EXISTS idx_registration_invites_student_id
    ON registration_invites (student_id)
    WHERE student_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_registration_invites_teacher_id
    ON registration_invites (teacher_id)
    WHERE teacher_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_registration_invites_active_student_unique
    ON registration_invites (student_id)
    WHERE student_id IS NOT NULL AND used_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_registration_invites_active_teacher_unique
    ON registration_invites (teacher_id)
    WHERE teacher_id IS NOT NULL AND used_at IS NULL;

CREATE OR REPLACE FUNCTION registration_invites_prepare_invite_code() RETURNS trigger AS $$
BEGIN
    IF NEW.invite_code IS NULL OR BTRIM(NEW.invite_code) = '' THEN
        RAISE EXCEPTION 'invite_code is required';
    END IF;

    NEW.invite_code := UPPER(BTRIM(NEW.invite_code));
    NEW.invite_code_hash := crypt(NEW.invite_code, gen_salt('bf', 10));
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_registration_invites_prepare_invite_code ON registration_invites;
CREATE TRIGGER trg_registration_invites_prepare_invite_code
BEFORE INSERT OR UPDATE OF invite_code ON registration_invites
FOR EACH ROW
EXECUTE FUNCTION registration_invites_prepare_invite_code();

INSERT INTO registration_invites (invite_code, role, student_id, used_at, created_at)
SELECT
    s.invite_code,
    'student'::user_role,
    s.student_id,
    s.invite_code_used_at,
    COALESCE(s.invite_code_used_at, NOW())
FROM students s
WHERE s.invite_code IS NOT NULL
ON CONFLICT (invite_code_hash) DO UPDATE
SET role = EXCLUDED.role,
    student_id = EXCLUDED.student_id,
    teacher_id = EXCLUDED.teacher_id,
    used_at = EXCLUDED.used_at;

SELECT setval(
    pg_get_serial_sequence('registration_invites', 'invite_id'),
    COALESCE((SELECT MAX(invite_id) FROM registration_invites), 1),
    true
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS trg_registration_invites_prepare_invite_code ON registration_invites;
DROP FUNCTION IF EXISTS registration_invites_prepare_invite_code();

DROP INDEX IF EXISTS idx_registration_invites_active_teacher_unique;
DROP INDEX IF EXISTS idx_registration_invites_active_student_unique;
DROP INDEX IF EXISTS idx_registration_invites_teacher_id;
DROP INDEX IF EXISTS idx_registration_invites_student_id;
DROP INDEX IF EXISTS idx_registration_invites_role;
DROP INDEX IF EXISTS idx_registration_invites_invite_code_hash_unique;
DROP INDEX IF EXISTS idx_registration_invites_invite_code_unique;

DROP TABLE IF EXISTS registration_invites;
-- +goose StatementEnd
