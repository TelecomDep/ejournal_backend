-- +goose Up
-- +goose StatementBegin
UPDATE students st
SET group_id = g.group_id
FROM users u, groups g
WHERE st.user_id = u.id
  AND u.login = 'student_test'
  AND g.group_name = 'TEST-GROUP-1';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE students st
SET group_id = NULL
FROM users u
WHERE st.user_id = u.id
  AND u.login = 'student_test';
-- +goose StatementEnd
