-- +goose Up
-- +goose StatementBegin
-- The Android demo teacher (login `teacher_test`) was previously seeded with a
-- single schedule row -> a single group, so `teacher_subjects` (which is scoped
-- to the teacher's schedule) only ever returned one group even though the
-- `groups` table holds many. This rebuilds the demo lesson so it covers every
-- group that actually has students, giving the demo teacher several groups.
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
    v_day_idx integer := EXTRACT(ISODOW FROM CURRENT_DATE)::integer;
    v_week_type smallint := CASE
        WHEN (EXTRACT(WEEK FROM CURRENT_DATE)::integer % 2 = 0) THEN 2
        ELSE 1
    END;
    v_parser_day_idx integer;
    v_inserted integer;
BEGIN
    -- `schedules` carries a BEFORE INSERT trigger that converts a parser day
    -- index (0..13) into the real ISO day_idx/week_type, so insert the parser
    -- value exactly like the other demo-schedule migrations do.
    IF v_week_type = 1 THEN
        v_parser_day_idx := v_day_idx - 1;
    ELSE
        v_parser_day_idx := v_day_idx + 6;
    END IF;

    SELECT t.teacher_id
    INTO v_teacher_id
    FROM teachers t
    JOIN users u ON u.id = t.user_id
    WHERE u.login = 'teacher_test'
    LIMIT 1;

    SELECT subject_id
    INTO v_subject_id
    FROM subjects
    WHERE subject_index = 'TEST-001'
    LIMIT 1;

    IF v_teacher_id IS NULL OR v_subject_id IS NULL THEN
        RAISE NOTICE 'skip demo multi-group schedule: teacher_id %, subject_id %', v_teacher_id, v_subject_id;
        RETURN;
    END IF;

    DELETE FROM schedules
    WHERE teacher_id = v_teacher_id
      AND lesson_num = 99;

    -- One schedule row per group that has at least one student, so every group
    -- the demo teacher sees also yields a usable attendance roster.
    INSERT INTO schedules (group_id, subject_id, teacher_id, lesson_num, day_idx, week_type, lesson_type)
    SELECT g.group_id, v_subject_id, v_teacher_id, 99, v_parser_day_idx, v_week_type, 'Практика'
    FROM groups g
    WHERE EXISTS (SELECT 1 FROM students s WHERE s.group_id = g.group_id);

    GET DIAGNOSTICS v_inserted = ROW_COUNT;

    -- Fallback: if no group has students yet, attach the first few groups so the
    -- demo teacher still has more than one group to choose from.
    IF v_inserted = 0 THEN
        INSERT INTO schedules (group_id, subject_id, teacher_id, lesson_num, day_idx, week_type, lesson_type)
        SELECT g.group_id, v_subject_id, v_teacher_id, 99, v_parser_day_idx, v_week_type, 'Практика'
        FROM groups g
        ORDER BY g.group_id
        LIMIT 5;
    END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Revert to a single demo group (mirrors the previous align migration).
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
    v_parser_day_idx integer;
BEGIN
    IF v_week_type = 1 THEN
        v_parser_day_idx := v_day_idx - 1;
    ELSE
        v_parser_day_idx := v_day_idx + 6;
    END IF;

    SELECT t.teacher_id
    INTO v_teacher_id
    FROM teachers t
    JOIN users u ON u.id = t.user_id
    WHERE u.login = 'teacher_test'
    LIMIT 1;

    SELECT subject_id
    INTO v_subject_id
    FROM subjects
    WHERE subject_index = 'TEST-001'
    LIMIT 1;

    SELECT group_id INTO v_group_id FROM groups WHERE group_id = 1 LIMIT 1;
    IF v_group_id IS NULL THEN
        SELECT group_id INTO v_group_id FROM groups ORDER BY group_id LIMIT 1;
    END IF;

    IF v_teacher_id IS NULL OR v_subject_id IS NULL OR v_group_id IS NULL THEN
        RETURN;
    END IF;

    DELETE FROM schedules
    WHERE teacher_id = v_teacher_id
      AND lesson_num = 99;

    INSERT INTO schedules (group_id, subject_id, teacher_id, lesson_num, day_idx, week_type, lesson_type)
    VALUES (v_group_id, v_subject_id, v_teacher_id, 99, v_parser_day_idx, v_week_type, 'Практика');
END $$;
-- +goose StatementEnd
