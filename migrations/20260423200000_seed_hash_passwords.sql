-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS pgcrypto;

UPDATE users
SET password_hash = crypt('8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92', gen_salt('bf', 10))
WHERE login IN ('teacher_test', 'student_test');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE users
SET password_hash = crypt('123456', gen_salt('bf', 10))
WHERE login IN ('teacher_test', 'student_test');
-- +goose StatementEnd
