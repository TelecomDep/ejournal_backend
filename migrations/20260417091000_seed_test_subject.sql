-- +goose Up
-- +goose StatementBegin
INSERT INTO subjects (subject_index, name, in_plan)
SELECT 'TEST-001', 'Networks', TRUE
WHERE NOT EXISTS (
    SELECT 1 FROM subjects WHERE subject_index = 'TEST-001'
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM subjects
WHERE subject_index = 'TEST-001';
-- +goose StatementEnd
