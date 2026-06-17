-- +goose Up
-- +goose StatementBegin
-- Demo showcase data for `student_test`: a full spring-2026 semester of
-- attendance (present / late / absent with timestamps so the heatmap lights up)
-- plus several graded works, so the dashboard looks alive when presenting the
-- service. All rows are tagged with a sentinel (sessions: created_at time
-- 07:07:07; grade items: title suffix " (демо)") so the Down migration can
-- remove exactly this seed and the Up migration is re-runnable.
DO $$
DECLARE
    v_teacher_id integer;
    v_subject_id integer;
    v_group_id   integer;
    v_student_id integer;
BEGIN
    SELECT t.teacher_id INTO v_teacher_id
    FROM teachers t JOIN users u ON u.id = t.user_id
    WHERE u.login = 'teacher_test' LIMIT 1;

    SELECT subject_id INTO v_subject_id
    FROM subjects WHERE subject_index = 'TEST-001' LIMIT 1;

    SELECT group_id INTO v_group_id
    FROM groups WHERE group_name = 'TEST-GROUP-1' LIMIT 1;

    SELECT st.student_id INTO v_student_id
    FROM students st JOIN users u ON u.id = st.user_id
    WHERE u.login = 'student_test' LIMIT 1;

    IF v_teacher_id IS NULL OR v_subject_id IS NULL OR v_group_id IS NULL OR v_student_id IS NULL THEN
        RAISE NOTICE 'skip student_test showcase seed: teacher %, subject %, group %, student %',
            v_teacher_id, v_subject_id, v_group_id, v_student_id;
        RETURN;
    END IF;

    --------------------------------------------------------------------------
    -- 1. Attendance: one lesson every 4 days across the spring semester.
    --------------------------------------------------------------------------
    -- Idempotency: drop any previous demo sessions (cascades to groups+students).
    DELETE FROM attendance_sessions
    WHERE teacher_id = v_teacher_id
      AND subject_id = v_subject_id
      AND created_at::time = TIME '07:07:07';

    INSERT INTO attendance_sessions (teacher_id, subject_id, expires_at, created_at)
    SELECT v_teacher_id, v_subject_id,
           (d + TIME '11:30'),
           (d + TIME '07:07:07')
    FROM generate_series(DATE '2026-02-09', DATE '2026-06-09', INTERVAL '4 days') AS d;

    INSERT INTO attendance_session_groups (session_id, group_id)
    SELECT s.session_id, v_group_id
    FROM attendance_sessions s
    WHERE s.teacher_id = v_teacher_id
      AND s.subject_id = v_subject_id
      AND s.created_at::time = TIME '07:07:07';

    -- Per-student status mix: mostly present, ~1/9 absent, a few late. The mix
    -- is deterministic (day-of-year + student_id) so it varies across the grid.
    INSERT INTO attendance_session_students (session_id, student_id, group_id_snapshot, status, marked_at, marked_by)
    SELECT s.session_id, st.student_id, v_group_id,
           CASE
               WHEN (EXTRACT(DOY FROM s.created_at)::int + st.student_id) % 9 = 0 THEN 'absent'
               WHEN (EXTRACT(DOY FROM s.created_at)::int + st.student_id) % 13 = 0 THEN 'late'
               ELSE 'present'
           END,
           CASE
               WHEN (EXTRACT(DOY FROM s.created_at)::int + st.student_id) % 9 = 0 THEN NULL
               ELSE s.created_at + INTERVAL '4 hours 23 minutes'
           END,
           CASE
               WHEN (EXTRACT(DOY FROM s.created_at)::int + st.student_id) % 9 = 0 THEN 'teacher'
               ELSE 'self'
           END
    FROM attendance_sessions s
    JOIN students st ON st.group_id = v_group_id
    WHERE s.teacher_id = v_teacher_id
      AND s.subject_id = v_subject_id
      AND s.created_at::time = TIME '07:07:07';

    --------------------------------------------------------------------------
    -- 2. Grades: a handful of graded works (БРС) on the subject. Total max of
    --    all subject items stays under the 100-point cap enforced by trigger.
    --------------------------------------------------------------------------
    -- Idempotency: drop previous demo grade items (cascades to their grades).
    DELETE FROM grade_items
    WHERE subject_id = v_subject_id
      AND title LIKE '%(демо)';

    INSERT INTO grade_items (subject_id, title, max_score, item_type, deadline) VALUES
        (v_subject_id, 'Тест №1 (демо)',          10, 'test',       TIMESTAMPTZ '2026-03-10 12:00:00+07'),
        (v_subject_id, 'Лабораторная №2 (демо)',  15, 'laboratory', TIMESTAMPTZ '2026-04-15 12:00:00+07'),
        (v_subject_id, 'Коллоквиум (демо)',       10, 'current',    TIMESTAMPTZ '2026-05-12 12:00:00+07'),
        (v_subject_id, 'Курсовой проект (демо)',  20, 'project',    TIMESTAMPTZ '2026-06-30 12:00:00+07');

    -- Grade student_test on the works whose deadline has already passed; the
    -- future "Курсовой проект" stays ungraded to show remaining headroom.
    INSERT INTO grades (student_id, item_id, teacher_id, score)
    SELECT v_student_id, gi.item_id, v_teacher_id,
           CASE gi.title
               WHEN 'Тест №1 (демо)'         THEN 9
               WHEN 'Лабораторная №2 (демо)' THEN 12
               WHEN 'Коллоквиум (демо)'      THEN 7
           END
    FROM grade_items gi
    WHERE gi.subject_id = v_subject_id
      AND gi.title IN ('Тест №1 (демо)', 'Лабораторная №2 (демо)', 'Коллоквиум (демо)')
    ON CONFLICT (student_id, item_id) DO UPDATE SET score = EXCLUDED.score;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$
DECLARE
    v_teacher_id integer;
    v_subject_id integer;
BEGIN
    SELECT t.teacher_id INTO v_teacher_id
    FROM teachers t JOIN users u ON u.id = t.user_id
    WHERE u.login = 'teacher_test' LIMIT 1;

    SELECT subject_id INTO v_subject_id
    FROM subjects WHERE subject_index = 'TEST-001' LIMIT 1;

    IF v_subject_id IS NOT NULL THEN
        DELETE FROM grade_items
        WHERE subject_id = v_subject_id
          AND title LIKE '%(демо)';
    END IF;

    IF v_teacher_id IS NOT NULL AND v_subject_id IS NOT NULL THEN
        DELETE FROM attendance_sessions
        WHERE teacher_id = v_teacher_id
          AND subject_id = v_subject_id
          AND created_at::time = TIME '07:07:07';
    END IF;
END $$;
-- +goose StatementEnd
