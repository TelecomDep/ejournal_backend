-- +goose Up
-- +goose StatementBegin
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

    SELECT group_id
    INTO v_group_id
    FROM groups
    WHERE group_id = 1
    LIMIT 1;

    IF v_group_id IS NULL THEN
        SELECT group_id INTO v_group_id
        FROM groups
        ORDER BY group_id
        LIMIT 1;
    END IF;

    IF v_teacher_id IS NULL OR v_subject_id IS NULL OR v_group_id IS NULL THEN
        RAISE NOTICE 'skip schedule align: teacher_id %, subject_id %, group_id %', v_teacher_id, v_subject_id, v_group_id;
        RETURN;
    END IF;

    DELETE FROM schedules
    WHERE teacher_id = v_teacher_id
      AND lesson_num = 99;

    INSERT INTO schedules (group_id, subject_id, teacher_id, lesson_num, day_idx, week_type, lesson_type)
    VALUES (v_group_id, v_subject_id, v_teacher_id, 99, v_parser_day_idx, v_week_type, 'Практика');
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM schedules
WHERE lesson_num = 99;
-- +goose StatementEnd
