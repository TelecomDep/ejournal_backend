-- +goose Up
-- +goose StatementBegin

-- A stream/cohort is optional because the source schedule does not always
-- provide one. Groups without an assigned stream remain visible as
-- "Без потока" in staff analytics.
ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS stream_name VARCHAR(100);

CREATE INDEX IF NOT EXISTS idx_groups_stream_name
    ON groups (stream_name);

-- Keep the presentation dataset useful immediately after migration.
UPDATE groups
SET stream_name = 'Демонстрационный поток 2026'
WHERE group_name IN ('DEMO-101', 'DEMO-102')
  AND (stream_name IS NULL OR BTRIM(stream_name) = '');

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_groups_stream_name;
ALTER TABLE groups DROP COLUMN IF EXISTS stream_name;
-- +goose StatementEnd
