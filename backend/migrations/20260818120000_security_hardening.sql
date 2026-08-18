-- +goose Up

-- Roles used by the admin API.
ALTER TYPE user_role ADD VALUE IF NOT EXISTS 'secretary';
ALTER TYPE user_role ADD VALUE IF NOT EXISTS 'program_creator';
ALTER TYPE user_role ADD VALUE IF NOT EXISTS 'director';
ALTER TYPE user_role ADD VALUE IF NOT EXISTS 'minister';

-- Increment this value to revoke the user's active sessions.
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS token_version BIGINT NOT NULL DEFAULT 1;

-- One-time codes for email changes and 2FA setup.
CREATE TABLE IF NOT EXISTS auth_challenges (
    challenge_id BIGSERIAL PRIMARY KEY,
    user_id      INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    purpose      VARCHAR(32) NOT NULL CHECK (purpose IN ('email_bind', '2fa_enable')),
    target       TEXT NOT NULL DEFAULT '',
    code_hash    BYTEA NOT NULL,
    attempts     SMALLINT NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    expires_at   TIMESTAMPTZ NOT NULL,
    consumed_at  TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS auth_challenges_active_idx
    ON auth_challenges (user_id, purpose, created_at DESC)
    WHERE consumed_at IS NULL;

-- Keep consent history after the account is deleted.
ALTER TABLE user_agreement_decisions
    DROP CONSTRAINT IF EXISTS user_agreement_decisions_user_id_fkey;

ALTER TABLE user_agreement_decisions
    ALTER COLUMN user_id DROP NOT NULL,
    ADD COLUMN IF NOT EXISTS actor_login TEXT,
    ADD COLUMN IF NOT EXISTS actor_role TEXT;

UPDATE user_agreement_decisions d
SET actor_login = COALESCE(d.actor_login, u.login, 'deleted-user'),
    actor_role = COALESCE(d.actor_role, u.role::text, 'unknown'),
    document_hash = COALESCE(NULLIF(d.document_hash, ''), 'legacy-unknown')
FROM users u
WHERE u.id = d.user_id;

UPDATE user_agreement_decisions
SET actor_login = COALESCE(actor_login, 'deleted-user'),
    actor_role = COALESCE(actor_role, 'unknown'),
    document_hash = COALESCE(NULLIF(document_hash, ''), 'legacy-unknown');

ALTER TABLE user_agreement_decisions
    ALTER COLUMN actor_login SET NOT NULL,
    ALTER COLUMN actor_role SET NOT NULL,
    ALTER COLUMN document_hash SET NOT NULL;

ALTER TABLE user_agreement_decisions
    ADD CONSTRAINT user_agreement_decisions_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL;

-- +goose Down

DROP TABLE IF EXISTS auth_challenges;

ALTER TABLE user_agreement_decisions
    DROP CONSTRAINT IF EXISTS user_agreement_decisions_user_id_fkey;

DELETE FROM user_agreement_decisions WHERE user_id IS NULL;

ALTER TABLE user_agreement_decisions
    ALTER COLUMN user_id SET NOT NULL,
    ALTER COLUMN document_hash DROP NOT NULL,
    DROP COLUMN IF EXISTS actor_login,
    DROP COLUMN IF EXISTS actor_role;

ALTER TABLE user_agreement_decisions
    ADD CONSTRAINT user_agreement_decisions_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE users DROP COLUMN IF EXISTS token_version;

-- Enum values stay in place: PostgreSQL has no safe DROP VALUE operation.
