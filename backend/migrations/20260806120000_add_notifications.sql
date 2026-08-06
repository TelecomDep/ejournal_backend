-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS notifications (
    notification_id BIGSERIAL PRIMARY KEY,

    category VARCHAR(32) NOT NULL,
    event_type VARCHAR(64) NOT NULL,

    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,

    created_by_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,

    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT notifications_category_check
        CHECK (category IN ('grades', 'schedule', 'attendance', 'system'))
);


CREATE TABLE IF NOT EXISTS notification_recipients (
    notification_id BIGINT NOT NULL
        REFERENCES notifications(notification_id) ON DELETE CASCADE,

    user_id INTEGER NOT NULL
        REFERENCES users(id) ON DELETE CASCADE,

    read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (notification_id, user_id)
);


CREATE TABLE IF NOT EXISTS notification_settings (
    user_id INTEGER PRIMARY KEY
        REFERENCES users(id) ON DELETE CASCADE,

    grades BOOLEAN NOT NULL DEFAULT TRUE,
    schedule BOOLEAN NOT NULL DEFAULT TRUE,
    attendance BOOLEAN NOT NULL DEFAULT TRUE,
    system BOOLEAN NOT NULL DEFAULT FALSE,

    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);


CREATE INDEX IF NOT EXISTS idx_notification_recipients_user_read
    ON notification_recipients (user_id, read_at);

CREATE INDEX IF NOT EXISTS idx_notifications_category_created
    ON notifications (category, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_notifications_expires_at
    ON notifications (expires_at)
    WHERE expires_at IS NOT NULL;
-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_notifications_expires_at;
DROP INDEX IF EXISTS idx_notifications_category_created;
DROP INDEX IF EXISTS idx_notification_recipients_user_read;

DROP TABLE IF EXISTS notification_settings;
DROP TABLE IF EXISTS notification_recipients;
DROP TABLE IF EXISTS notifications;
-- +goose StatementEnd
