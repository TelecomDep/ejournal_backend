-- +goose Up
-- SQL in this section is executed when the migration is applied.

CREATE TABLE IF NOT EXISTS user_avatars (
    user_id INT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    image_data BYTEA NOT NULL,
    content_type VARCHAR(64) NOT NULL DEFAULT 'image/png',
    hash VARCHAR(64) NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS attachments (
    id BIGSERIAL PRIMARY KEY,
    owner_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    filename VARCHAR(255) NOT NULL,
    file_size BIGINT NOT NULL,
    mime_type VARCHAR(128) NOT NULL,
    storage_type VARCHAR(32) NOT NULL DEFAULT 'db', -- 'db', 'disk', 's3'
    data BYTEA,                                    -- For inline preview / small files
    storage_path TEXT,                             -- For large files (disk path / s3 object key)
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_attachments_owner ON attachments(owner_id);

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.

DROP TABLE IF EXISTS attachments;
DROP TABLE IF EXISTS user_avatars;
