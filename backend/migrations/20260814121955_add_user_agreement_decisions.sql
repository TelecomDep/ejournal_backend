-- +goose Up

CREATE TABLE user_agreement_decisions (
    decision_id   BIGSERIAL PRIMARY KEY,
    user_id       INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    agreement_key TEXT NOT NULL DEFAULT 'user_agreement',
    version       TEXT NOT NULL,
    decision      TEXT NOT NULL CHECK (decision IN ('accepted', 'declined')),
    decided_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    document_hash TEXT,
    ip            INET,
    user_agent    TEXT
);

CREATE INDEX user_agreement_decisions_user_idx
    ON user_agreement_decisions (
        user_id,
        agreement_key,
        version,
        decided_at DESC,
        decision_id DESC
    );

-- +goose Down

DROP TABLE user_agreement_decisions;
