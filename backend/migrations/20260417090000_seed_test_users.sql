-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS pgcrypto;

INSERT INTO users (login, password_hash, role)
VALUES ('teacher_test', crypt('123456', gen_salt('bf', 10)), 'teacher')
ON CONFLICT (login) DO NOTHING;

INSERT INTO teachers (teacher_id, role, name)
SELECT id, 'teacher', 'Test Teacher'
FROM users
WHERE login = 'teacher_test' AND role = 'teacher'
ON CONFLICT (teacher_id) DO NOTHING;

INSERT INTO users (login, password_hash, role)
VALUES ('student_test', crypt('123456', gen_salt('bf', 10)), 'student')
ON CONFLICT (login) DO NOTHING;

INSERT INTO students (student_id, role, student_name)
SELECT id, 'student', 'Test Student'
FROM users
WHERE login = 'student_test' AND role = 'student'
ON CONFLICT (student_id) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM users
WHERE login IN ('teacher_test', 'student_test');
-- +goose StatementEnd
