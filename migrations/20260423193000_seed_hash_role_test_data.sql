-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS pgcrypto;

INSERT INTO groups (group_name)
SELECT 'TEST-GROUP-1'
WHERE NOT EXISTS (
    SELECT 1
    FROM groups
    WHERE group_name = 'TEST-GROUP-1'
);

INSERT INTO users (login, password_hash, role)
VALUES ('teacher_test', crypt('123456', gen_salt('bf', 10)), 'teacher')
ON CONFLICT (login) DO NOTHING;

INSERT INTO teachers (teacher_id, role, user_id, name)
SELECT u.id, 'teacher', u.id, 'Test Teacher'
FROM users u
WHERE u.login = 'teacher_test'
  AND u.role = 'teacher'
ON CONFLICT (teacher_id) DO NOTHING;

INSERT INTO subjects (subject_index, name, in_plan)
SELECT 'TEST-001', 'Networks', TRUE
WHERE NOT EXISTS (
    SELECT 1
    FROM subjects
    WHERE subject_index = 'TEST-001'
);

INSERT INTO lesson_times (lesson_num, start_time, end_time)
VALUES (
    99,
    (CURRENT_TIME + INTERVAL '5 minutes')::time,
    (CURRENT_TIME + INTERVAL '95 minutes')::time
)
ON CONFLICT (lesson_num) DO UPDATE
SET start_time = EXCLUDED.start_time,
    end_time = EXCLUDED.end_time;

DO $$
DECLARE
    v_teacher_id integer;
    v_subject_id integer;
    v_group_id integer;
    v_day_idx integer := EXTRACT(ISODOW FROM CURRENT_DATE)::integer;
    v_week_type smallint := CASE
        WHEN (EXTRACT(WEEK FROM CURRENT_DATE)::integer % 2 = 0) THEN 2
        ELSE 1
    END;
BEGIN
    SELECT t.teacher_id
    INTO v_teacher_id
    FROM teachers t
    JOIN users u ON u.id = t.user_id
    WHERE u.login = 'teacher_test'
    LIMIT 1;

    IF v_teacher_id IS NULL THEN
        SELECT teacher_id INTO v_teacher_id
        FROM teachers
        ORDER BY teacher_id
        LIMIT 1;
    END IF;

    SELECT subject_id
    INTO v_subject_id
    FROM subjects
    WHERE subject_index = 'TEST-001'
    LIMIT 1;

    IF v_subject_id IS NULL THEN
        SELECT subject_id INTO v_subject_id
        FROM subjects
        ORDER BY subject_id
        LIMIT 1;
    END IF;

    SELECT group_id
    INTO v_group_id
    FROM groups
    WHERE group_name = 'TEST-GROUP-1'
    LIMIT 1;

    IF v_group_id IS NULL THEN
        SELECT group_id INTO v_group_id
        FROM groups
        ORDER BY group_id
        LIMIT 1;
    END IF;

    IF v_teacher_id IS NULL OR v_subject_id IS NULL OR v_group_id IS NULL THEN
        RAISE NOTICE 'skip test schedule seed: teacher_id %, subject_id %, group_id %', v_teacher_id, v_subject_id, v_group_id;
        RETURN;
    END IF;

    DELETE FROM schedules
    WHERE teacher_id = v_teacher_id
      AND lesson_num = 99
      AND day_idx = v_day_idx
      AND week_type = v_week_type;

    INSERT INTO schedules (group_id, subject_id, teacher_id, lesson_num, day_idx, week_type, lesson_type)
    VALUES (v_group_id, v_subject_id, v_teacher_id, 99, v_day_idx, v_week_type, 'Практика');
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM schedules
WHERE lesson_num = 99;

DELETE FROM lesson_times
WHERE lesson_num = 99;
-- +goose StatementEnd
