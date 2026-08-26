-- +goose Up
-- +goose StatementBegin
-- Synchronize all sequences before seeding data so that auto-increment / serial / identity
-- primary keys do not conflict with existing records.
DO $$
DECLARE
    r RECORD;
    v_max BIGINT;
    v_seq TEXT;
BEGIN
    FOR r IN (
        SELECT table_name, column_name
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND (column_default LIKE 'nextval(%' OR is_identity = 'YES')
    ) LOOP
        v_seq := pg_get_serial_sequence(quote_ident(r.table_name), quote_ident(r.column_name));
        IF v_seq IS NOT NULL THEN
            EXECUTE format('SELECT COALESCE(MAX(%I), 0) FROM %I', r.column_name, r.table_name) INTO v_max;
            IF v_max > 0 THEN
                EXECUTE format('SELECT setval(%L, %s, true)', v_seq, v_max);
            END IF;
        END IF;
    END LOOP;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
-- Presentation semester: close the oversized legacy spring semester and open
-- an actual autumn 2026/2027 semester that contains the presentation date.
ALTER TABLE semesters DISABLE TRIGGER trg_semesters_validate_range;

DO $$
DECLARE
    v_actor_user_id INTEGER;
    v_target_semester_id INTEGER;
BEGIN
    SELECT id
    INTO v_actor_user_id
    FROM users
    WHERE login = 'admin_test'
    LIMIT 1;

    UPDATE semesters
    SET status = 'closed',
        is_current = FALSE,
        closed_at = COALESCE(closed_at, NOW()),
        closed_by_user_id = COALESCE(closed_by_user_id, v_actor_user_id)
    WHERE status = 'open'
      AND NOT (academic_year = '2026/2027' AND term_num = 1);

    -- The original seed accidentally extended spring 2025/2026 until June
    -- 2027. Shorten it before creating the next non-overlapping semester.
    UPDATE semesters
    SET ends_at = TIMESTAMPTZ '2026-07-31 23:59:59+07',
        status = 'closed',
        is_current = FALSE,
        closed_at = COALESCE(closed_at, TIMESTAMPTZ '2026-07-31 23:59:59+07'),
        closed_by_user_id = COALESCE(closed_by_user_id, v_actor_user_id)
    WHERE academic_year = '2025/2026'
      AND term_num = 2
      AND ends_at > TIMESTAMPTZ '2026-07-31 23:59:59+07';

    SELECT semester_id
    INTO v_target_semester_id
    FROM semesters
    WHERE academic_year = '2026/2027'
      AND term_num = 1
    LIMIT 1;

    IF v_target_semester_id IS NOT NULL THEN
        UPDATE semesters
        SET name = '2026/2027, осенний семестр — презентация',
            starts_at = TIMESTAMPTZ '2026-08-01 00:00:00+07',
            ends_at = TIMESTAMPTZ '2027-01-31 23:59:59+07',
            status = 'open',
            is_current = TRUE,
            opened_at = COALESCE(opened_at, NOW()),
            opened_by_user_id = COALESCE(opened_by_user_id, v_actor_user_id),
            closed_at = NULL,
            closed_by_user_id = NULL,
            archived_at = NULL,
            archived_by_user_id = NULL
        WHERE semester_id = v_target_semester_id;
    ELSE
        INSERT INTO semesters (
            academic_year,
            term_num,
            name,
            starts_at,
            ends_at,
            is_current,
            status,
            created_by_user_id,
            opened_at,
            opened_by_user_id
        )
        VALUES (
            '2026/2027',
            1,
            '2026/2027, осенний семестр — презентация',
            TIMESTAMPTZ '2026-08-01 00:00:00+07',
            TIMESTAMPTZ '2027-01-31 23:59:59+07',
            TRUE,
            'open',
            v_actor_user_id,
            NOW(),
            v_actor_user_id
        );
    END IF;
END $$;

ALTER TABLE semesters ENABLE TRIGGER trg_semesters_validate_range;
-- +goose StatementEnd

-- +goose StatementBegin
-- Deterministic accounts and organization scope used during the presentation.
DO $$
DECLARE
    v_faculty_id INTEGER;
    v_lectern_id INTEGER;
    v_group_101_id INTEGER;
    v_group_102_id INTEGER;
    v_teacher_user_id INTEGER;
    v_teacher_2_user_id INTEGER;
    v_teacher_id INTEGER;
    v_teacher_2_id INTEGER;
    v_student_user_id INTEGER;
    v_student_2_user_id INTEGER;
    v_student_3_user_id INTEGER;
    v_student_4_user_id INTEGER;
    v_student_5_user_id INTEGER;
    v_dean_user_id INTEGER;
BEGIN
    INSERT INTO faculties (code, name)
    VALUES ('DEMO-FIT', 'Презентационный факультет цифровых технологий')
    ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name
    RETURNING faculty_id INTO v_faculty_id;

    INSERT INTO lecterns (code, name, faculty_id)
    VALUES ('DEMO-IT', 'Презентационная кафедра информационных систем', v_faculty_id)
    ON CONFLICT (code) DO UPDATE
    SET name = EXCLUDED.name,
        faculty_id = EXCLUDED.faculty_id
    RETURNING lectern_id INTO v_lectern_id;

    INSERT INTO groups (group_name, lectern_id)
    VALUES ('DEMO-101', v_lectern_id)
    ON CONFLICT (group_name) DO UPDATE SET lectern_id = EXCLUDED.lectern_id
    RETURNING group_id INTO v_group_101_id;

    INSERT INTO groups (group_name, lectern_id)
    VALUES ('DEMO-102', v_lectern_id)
    ON CONFLICT (group_name) DO UPDATE SET lectern_id = EXCLUDED.lectern_id
    RETURNING group_id INTO v_group_102_id;

    INSERT INTO users (login, password_hash, role, status, is_2fa_enabled)
    VALUES
        ('teacher_test', crypt('123456', gen_salt('bf', 10)), 'teacher', 'active', FALSE),
        ('teacher_demo_2', crypt('123456', gen_salt('bf', 10)), 'teacher', 'active', FALSE),
        ('student_test', crypt('123456', gen_salt('bf', 10)), 'student', 'active', FALSE),
        ('student_demo_2', crypt('123456', gen_salt('bf', 10)), 'student', 'active', FALSE),
        ('student_demo_3', crypt('123456', gen_salt('bf', 10)), 'student', 'active', FALSE),
        ('student_demo_4', crypt('123456', gen_salt('bf', 10)), 'student', 'active', FALSE),
        ('student_demo_5', crypt('123456', gen_salt('bf', 10)), 'student', 'active', FALSE),
        ('dean_test', crypt('123456', gen_salt('bf', 10)), 'dean', 'active', FALSE)
    ON CONFLICT (login) DO UPDATE
    SET password_hash = EXCLUDED.password_hash,
        role = EXCLUDED.role,
        status = 'active',
        is_2fa_enabled = FALSE,
        totp_secret = NULL,
        token_version = users.token_version + 1;

    SELECT id INTO v_teacher_user_id FROM users WHERE login = 'teacher_test';
    SELECT id INTO v_teacher_2_user_id FROM users WHERE login = 'teacher_demo_2';
    SELECT id INTO v_student_user_id FROM users WHERE login = 'student_test';
    SELECT id INTO v_student_2_user_id FROM users WHERE login = 'student_demo_2';
    SELECT id INTO v_student_3_user_id FROM users WHERE login = 'student_demo_3';
    SELECT id INTO v_student_4_user_id FROM users WHERE login = 'student_demo_4';
    SELECT id INTO v_student_5_user_id FROM users WHERE login = 'student_demo_5';
    SELECT id INTO v_dean_user_id FROM users WHERE login = 'dean_test';

    UPDATE teachers
    SET user_id = v_teacher_user_id,
        role = 'teacher',
        name = 'Тестовый Преподаватель',
        lectern_id = v_lectern_id,
        job_title = 'Доцент'
    WHERE user_id = v_teacher_user_id
       OR (user_id IS NULL AND teacher_id = v_teacher_user_id);

    IF NOT FOUND THEN
        INSERT INTO teachers (user_id, role, name, lectern_id, job_title)
        VALUES (v_teacher_user_id, 'teacher', 'Тестовый Преподаватель', v_lectern_id, 'Доцент');
    END IF;

    UPDATE teachers
    SET user_id = v_teacher_2_user_id,
        role = 'teacher',
        name = 'Демонстрационный Ассистент',
        lectern_id = v_lectern_id,
        job_title = 'Ассистент'
    WHERE user_id = v_teacher_2_user_id
       OR (user_id IS NULL AND teacher_id = v_teacher_2_user_id);

    IF NOT FOUND THEN
        INSERT INTO teachers (user_id, role, name, lectern_id, job_title)
        VALUES (v_teacher_2_user_id, 'teacher', 'Демонстрационный Ассистент', v_lectern_id, 'Ассистент');
    END IF;

    SELECT teacher_id INTO v_teacher_id FROM teachers WHERE user_id = v_teacher_user_id LIMIT 1;
    SELECT teacher_id INTO v_teacher_2_id FROM teachers WHERE user_id = v_teacher_2_user_id LIMIT 1;

    UPDATE students
    SET user_id = v_student_user_id,
        role = 'student',
        student_name = 'Тестовый Студент',
        group_id = v_group_101_id,
        total_cheat_attempts = 0
    WHERE user_id = v_student_user_id
       OR (user_id IS NULL AND student_id = v_student_user_id);
    IF NOT FOUND THEN
        INSERT INTO students (user_id, role, student_name, group_id, total_cheat_attempts)
        VALUES (v_student_user_id, 'student', 'Тестовый Студент', v_group_101_id, 0);
    END IF;

    UPDATE students
    SET user_id = v_student_2_user_id,
        role = 'student',
        student_name = 'Анна Смирнова',
        group_id = v_group_101_id,
        total_cheat_attempts = 2
    WHERE user_id = v_student_2_user_id
       OR (user_id IS NULL AND student_id = v_student_2_user_id);
    IF NOT FOUND THEN
        INSERT INTO students (user_id, role, student_name, group_id, total_cheat_attempts)
        VALUES (v_student_2_user_id, 'student', 'Анна Смирнова', v_group_101_id, 2);
    END IF;

    UPDATE students
    SET user_id = v_student_3_user_id,
        role = 'student',
        student_name = 'Иван Волков',
        group_id = v_group_101_id,
        total_cheat_attempts = 0
    WHERE user_id = v_student_3_user_id
       OR (user_id IS NULL AND student_id = v_student_3_user_id);
    IF NOT FOUND THEN
        INSERT INTO students (user_id, role, student_name, group_id, total_cheat_attempts)
        VALUES (v_student_3_user_id, 'student', 'Иван Волков', v_group_101_id, 0);
    END IF;

    UPDATE students
    SET user_id = v_student_4_user_id,
        role = 'student',
        student_name = 'Мария Орлова',
        group_id = v_group_102_id,
        total_cheat_attempts = 0
    WHERE user_id = v_student_4_user_id
       OR (user_id IS NULL AND student_id = v_student_4_user_id);
    IF NOT FOUND THEN
        INSERT INTO students (user_id, role, student_name, group_id, total_cheat_attempts)
        VALUES (v_student_4_user_id, 'student', 'Мария Орлова', v_group_102_id, 0);
    END IF;

    UPDATE students
    SET user_id = v_student_5_user_id,
        role = 'student',
        student_name = 'Павел Соколов',
        group_id = v_group_102_id,
        total_cheat_attempts = 0
    WHERE user_id = v_student_5_user_id
       OR (user_id IS NULL AND student_id = v_student_5_user_id);
    IF NOT FOUND THEN
        INSERT INTO students (user_id, role, student_name, group_id, total_cheat_attempts)
        VALUES (v_student_5_user_id, 'student', 'Павел Соколов', v_group_102_id, 0);
    END IF;

    DELETE FROM org_scopes WHERE user_id = v_dean_user_id;
    INSERT INTO org_scopes (user_id, faculty_id)
    VALUES (v_dean_user_id, v_faculty_id);

    INSERT INTO notification_settings (user_id, grades, schedule, attendance, system)
    SELECT id, TRUE, TRUE, TRUE, TRUE
    FROM users
    WHERE login IN (
        'teacher_test', 'teacher_demo_2', 'student_test', 'student_demo_2',
        'student_demo_3', 'student_demo_4', 'student_demo_5', 'dean_test'
    )
    ON CONFLICT (user_id) DO UPDATE
    SET grades = TRUE,
        schedule = TRUE,
        attendance = TRUE,
        system = TRUE,
        updated_at = NOW();

    INSERT INTO registration_invites (invite_code, invite_code_hash, role, teacher_id, used_at)
    SELECT 'PRESENT-TEACHER-01', '', 'teacher', v_teacher_id, NOW()
    WHERE NOT EXISTS (SELECT 1 FROM registration_invites WHERE teacher_id = v_teacher_id);

    INSERT INTO registration_invites (invite_code, invite_code_hash, role, teacher_id, used_at)
    SELECT 'PRESENT-TEACHER-02', '', 'teacher', v_teacher_2_id, NOW()
    WHERE NOT EXISTS (SELECT 1 FROM registration_invites WHERE teacher_id = v_teacher_2_id);

    INSERT INTO registration_invites (invite_code, invite_code_hash, role, student_id, used_at)
    SELECT 'PRESENT-STUDENT-01', '', 'student', st.student_id, NOW()
    FROM students st
    WHERE st.user_id = v_student_user_id
      AND NOT EXISTS (SELECT 1 FROM registration_invites ri WHERE ri.student_id = st.student_id);

    INSERT INTO registration_invites (invite_code, invite_code_hash, role, student_id, used_at)
    SELECT 'PRESENT-STUDENT-' || LPAD((ROW_NUMBER() OVER (ORDER BY st.student_id) + 1)::TEXT, 2, '0'),
           '',
           'student',
           st.student_id,
           NOW()
    FROM students st
    WHERE st.user_id IN (v_student_2_user_id, v_student_3_user_id, v_student_4_user_id, v_student_5_user_id)
      AND NOT EXISTS (SELECT 1 FROM registration_invites ri WHERE ri.student_id = st.student_id);

    INSERT INTO user_agreement_decisions (
        user_id,
        agreement_key,
        version,
        decision,
        decided_at,
        document_hash,
        actor_login,
        actor_role,
        user_agent
    )
    SELECT u.id,
           'user_agreement',
           '2026-08-01',
           CASE WHEN u.login = 'student_demo_3' THEN 'declined' ELSE 'accepted' END,
           NOW(),
           encode(digest('user_agreement' || E'\n' || '2026-08-01' || E'\n' ||
                         'Я даю согласие на обработку персональных данных', 'sha256'), 'hex'),
           u.login,
           u.role::TEXT,
           'presentation-demo-migration'
    FROM users u
    WHERE u.login IN ('student_test', 'student_demo_2', 'student_demo_3', 'student_demo_4', 'student_demo_5');
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
-- Academic catalogue: three subjects are enough for the student radar chart,
-- while metrics and controls make the staff views and reports non-empty.
DO $$
DECLARE
    v_lectern_id INTEGER;
    v_semester_id INTEGER;
    v_subject_id INTEGER;
    v_control_type_id INTEGER;
    v_subject_index TEXT;
BEGIN
    SELECT lectern_id INTO v_lectern_id
    FROM lecterns
    WHERE code IN ('DEMO-IT', 'KAF-IT')
    ORDER BY CASE WHEN code = 'DEMO-IT' THEN 0 ELSE 1 END
    LIMIT 1;
    SELECT semester_id INTO v_semester_id
    FROM semesters
    WHERE academic_year = '2026/2027' AND term_num = 1;

    INSERT INTO subjects (subject_index, name, in_plan, lectern_id)
    SELECT 'DEMO-ALG', 'Алгоритмы и структуры данных', TRUE, v_lectern_id
    WHERE NOT EXISTS (SELECT 1 FROM subjects WHERE subject_index = 'DEMO-ALG');

    INSERT INTO subjects (subject_index, name, in_plan, lectern_id)
    SELECT 'DEMO-NET', 'Компьютерные сети', TRUE, v_lectern_id
    WHERE NOT EXISTS (SELECT 1 FROM subjects WHERE subject_index = 'DEMO-NET');

    INSERT INTO subjects (subject_index, name, in_plan, lectern_id)
    SELECT 'DEMO-DB', 'Базы данных', TRUE, v_lectern_id
    WHERE NOT EXISTS (SELECT 1 FROM subjects WHERE subject_index = 'DEMO-DB');

    UPDATE subjects
    SET lectern_id = v_lectern_id,
        in_plan = TRUE,
        name = CASE subject_index
            WHEN 'DEMO-ALG' THEN 'Алгоритмы и структуры данных'
            WHEN 'DEMO-NET' THEN 'Компьютерные сети'
            WHEN 'DEMO-DB' THEN 'Базы данных'
            ELSE name
        END
    WHERE subject_index IN ('DEMO-ALG', 'DEMO-NET', 'DEMO-DB');

    FOR v_subject_index IN SELECT unnest(ARRAY['DEMO-ALG', 'DEMO-NET', 'DEMO-DB'])
    LOOP
        SELECT subject_id INTO v_subject_id
        FROM subjects
        WHERE subject_index = v_subject_index
        ORDER BY subject_id
        LIMIT 1;

        INSERT INTO subject_metrics (
            subject_id,
            zet_expert,
            zet_fact,
            hours_expert,
            hours_by_plan,
            hours_contr_work,
            hours_auditory,
            hours_self_study,
            hours_control,
            hours_prep
        )
        VALUES (v_subject_id, 4, 4, 144, 144, 18, 64, 62, 8, 10)
        ON CONFLICT (subject_id) DO UPDATE
        SET zet_expert = EXCLUDED.zet_expert,
            zet_fact = EXCLUDED.zet_fact,
            hours_expert = EXCLUDED.hours_expert,
            hours_by_plan = EXCLUDED.hours_by_plan,
            hours_contr_work = EXCLUDED.hours_contr_work,
            hours_auditory = EXCLUDED.hours_auditory,
            hours_self_study = EXCLUDED.hours_self_study,
            hours_control = EXCLUDED.hours_control,
            hours_prep = EXCLUDED.hours_prep;

        INSERT INTO semester_load (subject_id, semester_num, semester_id, zet_value)
        SELECT v_subject_id, 1, v_semester_id, 4
        WHERE NOT EXISTS (
            SELECT 1
            FROM semester_load
            WHERE subject_id = v_subject_id
              AND semester_num = 1
              AND semester_id = v_semester_id
        );

        SELECT type_id INTO v_control_type_id
        FROM control_types
        WHERE type_name = CASE WHEN v_subject_index = 'DEMO-NET' THEN 'Зачет' ELSE 'Экзамен' END
        LIMIT 1;

        INSERT INTO subject_controls (subject_id, type_id, semester_num, semester_id)
        SELECT v_subject_id, v_control_type_id, 1, v_semester_id
        WHERE NOT EXISTS (
            SELECT 1
            FROM subject_controls
            WHERE subject_id = v_subject_id
              AND type_id = v_control_type_id
              AND semester_num = 1
              AND semester_id = v_semester_id
        );
    END LOOP;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
-- A complete two-week timetable (both week parities and every weekday) means
-- the test student and teacher always have lessons to show on presentation day.
INSERT INTO lesson_times (lesson_num, start_time, end_time)
VALUES
    (1, TIME '08:00', TIME '09:35'),
    (2, TIME '09:50', TIME '11:25'),
    (3, TIME '11:40', TIME '13:15')
ON CONFLICT (lesson_num) DO UPDATE
SET start_time = EXCLUDED.start_time,
    end_time = EXCLUDED.end_time;

DELETE FROM schedules sch
USING semesters sem, teachers t, users u, subjects sub, groups g
WHERE sch.semester_id = sem.semester_id
  AND sch.teacher_id = t.teacher_id
  AND t.user_id = u.id
  AND sch.subject_id = sub.subject_id
  AND sch.group_id = g.group_id
  AND sem.academic_year = '2026/2027'
  AND sem.term_num = 1
  AND u.login = 'teacher_test'
  AND sub.subject_index IN ('DEMO-ALG', 'DEMO-NET', 'DEMO-DB')
  AND g.group_name IN ('DEMO-101', 'DEMO-102');

INSERT INTO schedules (
    group_id,
    subject_id,
    teacher_id,
    lesson_num,
    day_idx,
    week_type,
    subgroup,
    lesson_type,
    room_info,
    semester_id
)
SELECT g.group_id,
       sub.subject_id,
       t.teacher_id,
       CASE sub.subject_index
           WHEN 'DEMO-ALG' THEN 1
           WHEN 'DEMO-NET' THEN 2
           ELSE 3
       END,
       raw_day.day_idx,
       1,
       'Вся группа',
       CASE sub.subject_index
           WHEN 'DEMO-ALG' THEN 'Лекция'
           WHEN 'DEMO-NET' THEN 'Лабораторная работа'
           ELSE 'Практика'
       END,
       CASE sub.subject_index
           WHEN 'DEMO-ALG' THEN 'Аудитория 401'
           WHEN 'DEMO-NET' THEN 'Лаборатория 305'
           ELSE 'Аудитория 212'
       END,
       sem.semester_id
FROM groups g
CROSS JOIN subjects sub
CROSS JOIN generate_series(0, 13) AS raw_day(day_idx)
JOIN users u ON u.login = 'teacher_test'
JOIN teachers t ON t.user_id = u.id
JOIN semesters sem ON sem.academic_year = '2026/2027' AND sem.term_num = 1
WHERE g.group_name IN ('DEMO-101', 'DEMO-102')
  AND sub.subject_index IN ('DEMO-ALG', 'DEMO-NET', 'DEMO-DB');
-- +goose StatementEnd

-- +goose StatementBegin
-- Attendance history, including teacher corrections, late/excused statuses and
-- one antifraud incident visible to the dean account.
ALTER TABLE attendance_sessions DISABLE TRIGGER trg_attendance_sessions_open_semester;
ALTER TABLE attendance_session_students DISABLE TRIGGER trg_attendance_students_open_semester;

DELETE FROM attendance_session_students ass
USING attendance_sessions sess, semesters sem, teachers t, users u
WHERE ass.session_id = sess.session_id
  AND sess.semester_id = sem.semester_id
  AND sess.teacher_id = t.teacher_id
  AND t.user_id = u.id
  AND sem.academic_year = '2026/2027'
  AND sem.term_num = 1
  AND u.login = 'teacher_test'
  AND sess.lesson_name LIKE '[DEMO]%';

DELETE FROM attendance_session_groups asg
USING attendance_sessions sess, semesters sem, teachers t, users u
WHERE asg.session_id = sess.session_id
  AND sess.semester_id = sem.semester_id
  AND sess.teacher_id = t.teacher_id
  AND t.user_id = u.id
  AND sem.academic_year = '2026/2027'
  AND sem.term_num = 1
  AND u.login = 'teacher_test'
  AND sess.lesson_name LIKE '[DEMO]%';

DELETE FROM attendance_sessions sess
USING semesters sem, teachers t, users u
WHERE sess.semester_id = sem.semester_id
  AND sess.teacher_id = t.teacher_id
  AND t.user_id = u.id
  AND sem.academic_year = '2026/2027'
  AND sem.term_num = 1
  AND u.login = 'teacher_test'
  AND sess.lesson_name LIKE '[DEMO]%';

WITH session_seed(subject_index, occurred_on, lesson_name) AS (
    VALUES
        ('DEMO-ALG', DATE '2026-08-10', '[DEMO] Алгоритмы — лекция 1'),
        ('DEMO-ALG', DATE '2026-08-17', '[DEMO] Алгоритмы — лекция 2'),
        ('DEMO-ALG', DATE '2026-08-24', '[DEMO] Алгоритмы — лекция 3'),
        ('DEMO-NET', DATE '2026-08-11', '[DEMO] Сети — лабораторная 1'),
        ('DEMO-NET', DATE '2026-08-18', '[DEMO] Сети — лабораторная 2'),
        ('DEMO-NET', DATE '2026-08-25', '[DEMO] Сети — лабораторная 3'),
        ('DEMO-DB', DATE '2026-08-12', '[DEMO] Базы данных — практика 1'),
        ('DEMO-DB', DATE '2026-08-19', '[DEMO] Базы данных — практика 2'),
        ('DEMO-DB', DATE '2026-08-26', '[DEMO] Базы данных — практика 3')
)
INSERT INTO attendance_sessions (
    teacher_id,
    subject_id,
    semester_id,
    lesson_name,
    lat,
    lon,
    expires_at,
    created_at
)
SELECT t.teacher_id,
       sub.subject_id,
       sem.semester_id,
       seed.lesson_name,
       55.041500,
       82.934600,
       (seed.occurred_on::TIMESTAMP + TIME '09:35') AT TIME ZONE 'Asia/Novosibirsk',
       (seed.occurred_on::TIMESTAMP + TIME '08:00') AT TIME ZONE 'Asia/Novosibirsk'
FROM session_seed seed
JOIN subjects sub ON sub.subject_index = seed.subject_index
JOIN users u ON u.login = 'teacher_test'
JOIN teachers t ON t.user_id = u.id
JOIN semesters sem ON sem.academic_year = '2026/2027' AND sem.term_num = 1;

INSERT INTO attendance_session_groups (session_id, group_id)
SELECT sess.session_id, g.group_id
FROM attendance_sessions sess
JOIN semesters sem ON sem.semester_id = sess.semester_id
CROSS JOIN groups g
WHERE sem.academic_year = '2026/2027'
  AND sem.term_num = 1
  AND sess.lesson_name LIKE '[DEMO]%'
  AND g.group_name IN ('DEMO-101', 'DEMO-102')
ON CONFLICT DO NOTHING;

WITH roster AS (
    SELECT sess.session_id,
           sess.created_at,
           sub.subject_index,
           st.student_id,
           st.group_id,
           u.login,
           DENSE_RANK() OVER (
               PARTITION BY sub.subject_index
               ORDER BY sess.created_at, sess.session_id
           ) AS session_no
    FROM attendance_sessions sess
    JOIN semesters sem ON sem.semester_id = sess.semester_id
    JOIN subjects sub ON sub.subject_id = sess.subject_id
    JOIN students st ON st.group_id IN (
        SELECT group_id FROM groups WHERE group_name IN ('DEMO-101', 'DEMO-102')
    )
    LEFT JOIN users u ON u.id = st.user_id
    WHERE sem.academic_year = '2026/2027'
      AND sem.term_num = 1
      AND sess.lesson_name LIKE '[DEMO]%'
), statuses AS (
    SELECT roster.*,
           CASE
               WHEN login = 'student_test' AND subject_index = 'DEMO-ALG' THEN
                   CASE session_no WHEN 1 THEN 'present' WHEN 2 THEN 'late' ELSE 'absent' END
               WHEN login = 'student_test' AND subject_index = 'DEMO-NET' THEN
                   CASE session_no WHEN 3 THEN 'late' ELSE 'present' END
               WHEN login = 'student_test' AND subject_index = 'DEMO-DB' THEN
                   CASE session_no WHEN 2 THEN 'excused' ELSE 'present' END
               WHEN MOD(student_id + session_no, 9) = 0 THEN 'absent'
               WHEN MOD(student_id + session_no, 7) = 0 THEN 'excused'
               WHEN MOD(student_id + session_no, 5) = 0 THEN 'late'
               ELSE 'present'
           END AS attendance_status,
           login = 'student_demo_2'
               AND subject_index = 'DEMO-NET'
               AND session_no = 2 AS fraud
    FROM roster
)
INSERT INTO attendance_session_students (
    session_id,
    student_id,
    group_id_snapshot,
    status,
    marked_at,
    marked_by,
    device_id,
    check_in_lat,
    check_in_lon,
    is_fraud,
    fraud_reason
)
SELECT session_id,
       student_id,
       group_id,
       attendance_status,
       created_at + INTERVAL '12 minutes',
       CASE WHEN attendance_status IN ('absent', 'excused') THEN 'teacher' ELSE 'self' END,
       CASE WHEN fraud THEN 'demo-shared-device-001' ELSE 'demo-device-' || student_id::TEXT END,
       CASE WHEN attendance_status IN ('present', 'late') THEN 55.041510 ELSE NULL END,
       CASE WHEN attendance_status IN ('present', 'late') THEN 82.934610 ELSE NULL END,
       fraud,
       CASE WHEN fraud THEN 'Повторная отметка с устройства другого студента' ELSE NULL END
FROM statuses
ON CONFLICT (session_id, student_id) DO UPDATE
SET status = EXCLUDED.status,
    marked_at = EXCLUDED.marked_at,
    marked_by = EXCLUDED.marked_by,
    device_id = EXCLUDED.device_id,
    check_in_lat = EXCLUDED.check_in_lat,
    check_in_lon = EXCLUDED.check_in_lon,
    is_fraud = EXCLUDED.is_fraud,
    fraud_reason = EXCLUDED.fraud_reason;

ALTER TABLE attendance_session_students ENABLE TRIGGER trg_attendance_students_open_semester;
ALTER TABLE attendance_sessions ENABLE TRIGGER trg_attendance_sessions_open_semester;
-- +goose StatementEnd

-- +goose StatementBegin
-- BRS grade book: every subject totals exactly 100 points and includes an
-- automatic attendance item, laboratory, practice, test and future project.
ALTER TABLE grade_items DISABLE TRIGGER trg_grade_items_open_semester;
ALTER TABLE grade_items DISABLE TRIGGER trg_limit_subject_scores;
ALTER TABLE grades DISABLE TRIGGER trg_grades_open_semester;
ALTER TABLE grades DISABLE TRIGGER trg_limit_grade_score;

DELETE FROM grade_events ge
USING grade_items gi, semesters sem, subjects sub
WHERE ge.grade_item_id = gi.item_id
  AND gi.semester_id = sem.semester_id
  AND gi.subject_id = sub.subject_id
  AND sem.academic_year = '2026/2027'
  AND sem.term_num = 1
  AND sub.subject_index IN ('DEMO-ALG', 'DEMO-NET', 'DEMO-DB')
  AND gi.title LIKE '[DEMO]%';

DELETE FROM grades gr
USING grade_items gi, semesters sem, subjects sub
WHERE gr.item_id = gi.item_id
  AND gi.semester_id = sem.semester_id
  AND gi.subject_id = sub.subject_id
  AND sem.academic_year = '2026/2027'
  AND sem.term_num = 1
  AND sub.subject_index IN ('DEMO-ALG', 'DEMO-NET', 'DEMO-DB')
  AND gi.title LIKE '[DEMO]%';

DELETE FROM grade_items gi
USING semesters sem, subjects sub
WHERE gi.semester_id = sem.semester_id
  AND gi.subject_id = sub.subject_id
  AND sem.academic_year = '2026/2027'
  AND sem.term_num = 1
  AND sub.subject_index IN ('DEMO-ALG', 'DEMO-NET', 'DEMO-DB')
  AND gi.title LIKE '[DEMO]%';

WITH item_seed(title, max_score, item_type, deadline, created_at) AS (
    VALUES
        ('[DEMO] Посещаемость', 10, 'attendance_auto', TIMESTAMPTZ '2026-08-25 23:59:00+07', TIMESTAMPTZ '2026-08-01 10:00:00+07'),
        ('[DEMO] Лабораторная работа', 20, 'laboratory', TIMESTAMPTZ '2026-08-12 23:59:00+07', TIMESTAMPTZ '2026-08-01 10:05:00+07'),
        ('[DEMO] Практическая работа', 20, 'practice', TIMESTAMPTZ '2026-08-19 23:59:00+07', TIMESTAMPTZ '2026-08-01 10:10:00+07'),
        ('[DEMO] Контрольный тест', 20, 'test', TIMESTAMPTZ '2026-08-25 23:59:00+07', TIMESTAMPTZ '2026-08-01 10:15:00+07'),
        ('[DEMO] Итоговый проект', 30, 'project', TIMESTAMPTZ '2026-10-15 23:59:00+07', TIMESTAMPTZ '2026-08-01 10:20:00+07')
)
INSERT INTO grade_items (
    subject_id,
    semester_id,
    created_by_teacher_id,
    title,
    max_score,
    item_type,
    deadline,
    created_at
)
SELECT sub.subject_id,
       sem.semester_id,
       t.teacher_id,
       seed.title,
       seed.max_score,
       seed.item_type,
       seed.deadline,
       seed.created_at
FROM subjects sub
CROSS JOIN item_seed seed
JOIN semesters sem ON sem.academic_year = '2026/2027' AND sem.term_num = 1
JOIN users u ON u.login = 'teacher_test'
JOIN teachers t ON t.user_id = u.id
WHERE sub.subject_index IN ('DEMO-ALG', 'DEMO-NET', 'DEMO-DB');

INSERT INTO grades (
    student_id,
    item_id,
    teacher_id,
    score,
    session_id,
    comment,
    created_at,
    updated_at
)
SELECT st.student_id,
       gi.item_id,
       t.teacher_id,
       CASE
           WHEN u_student.login = 'student_test' AND gi.item_type = 'attendance_auto' THEN
               CASE sub.subject_index WHEN 'DEMO-NET' THEN 10 ELSE 7 END
           WHEN u_student.login = 'student_test' AND gi.item_type = 'laboratory' THEN
               CASE sub.subject_index WHEN 'DEMO-DB' THEN 19 ELSE 18 END
           WHEN u_student.login = 'student_test' AND gi.item_type = 'practice' THEN
               CASE sub.subject_index WHEN 'DEMO-NET' THEN 18 ELSE 17 END
           WHEN u_student.login = 'student_test' AND gi.item_type = 'test' THEN
               CASE sub.subject_index WHEN 'DEMO-ALG' THEN 19 WHEN 'DEMO-NET' THEN 18 ELSE 20 END
           WHEN gi.item_type = 'attendance_auto' THEN GREATEST(5, 10 - MOD(st.student_id + sub.subject_id, 5))
           WHEN gi.item_type = 'laboratory' THEN GREATEST(10, 20 - MOD(st.student_id + sub.subject_id, 7))
           WHEN gi.item_type = 'practice' THEN GREATEST(10, 20 - MOD(st.student_id + sub.subject_id + 2, 8))
           ELSE GREATEST(10, 20 - MOD(st.student_id + sub.subject_id + 4, 9))
       END,
       linked_session.session_id,
       CASE
           WHEN gi.item_type = 'attendance_auto' THEN 'Автоматическая оценка за посещаемость'
           ELSE 'Демонстрационная оценка преподавателя'
       END,
       LEAST(NOW(), gi.deadline + INTERVAL '1 hour'),
       LEAST(NOW(), gi.deadline + INTERVAL '1 hour')
FROM grade_items gi
JOIN subjects sub ON sub.subject_id = gi.subject_id
JOIN semesters sem ON sem.semester_id = gi.semester_id
JOIN users u_teacher ON u_teacher.login = 'teacher_test'
JOIN teachers t ON t.user_id = u_teacher.id
CROSS JOIN students st
LEFT JOIN users u_student ON u_student.id = st.user_id
LEFT JOIN LATERAL (
    SELECT sess.session_id
    FROM attendance_sessions sess
    WHERE sess.subject_id = gi.subject_id
      AND sess.semester_id = gi.semester_id
    ORDER BY sess.created_at DESC, sess.session_id DESC
    LIMIT 1
) linked_session ON TRUE
WHERE sem.academic_year = '2026/2027'
  AND sem.term_num = 1
  AND sub.subject_index IN ('DEMO-ALG', 'DEMO-NET', 'DEMO-DB')
  AND gi.title LIKE '[DEMO]%'
  AND gi.item_type <> 'project'
  AND st.group_id IN (SELECT group_id FROM groups WHERE group_name IN ('DEMO-101', 'DEMO-102'))
ON CONFLICT (student_id, item_id) DO UPDATE
SET teacher_id = EXCLUDED.teacher_id,
    score = EXCLUDED.score,
    session_id = EXCLUDED.session_id,
    comment = EXCLUDED.comment,
    updated_at = EXCLUDED.updated_at,
    deleted_at = NULL,
    deleted_by_user_id = NULL,
    delete_reason = NULL;

INSERT INTO grade_events (
    grade_id,
    grade_item_id,
    student_id,
    actor_user_id,
    event_type,
    old_score,
    new_score,
    reason,
    created_at
)
SELECT NULL,
       gi.item_id,
       NULL,
       u.id,
       'grade_item_created',
       NULL,
       NULL,
       'Демонстрационная контрольная точка',
       gi.created_at
FROM grade_items gi
JOIN subjects sub ON sub.subject_id = gi.subject_id
JOIN semesters sem ON sem.semester_id = gi.semester_id
JOIN users u ON u.login = 'teacher_test'
WHERE sem.academic_year = '2026/2027'
  AND sem.term_num = 1
  AND sub.subject_index IN ('DEMO-ALG', 'DEMO-NET', 'DEMO-DB')
  AND gi.title LIKE '[DEMO]%'
  AND NOT EXISTS (
      SELECT 1 FROM grade_events existing
      WHERE existing.grade_item_id = gi.item_id
        AND existing.event_type = 'grade_item_created'
  );

INSERT INTO grade_events (
    grade_id,
    grade_item_id,
    student_id,
    actor_user_id,
    event_type,
    old_score,
    new_score,
    reason,
    created_at
)
SELECT gr.grade_id,
       gr.item_id,
       gr.student_id,
       u.id,
       'grade_created',
       NULL,
       gr.score,
       'Демонстрационная оценка',
       gr.created_at
FROM grades gr
JOIN grade_items gi ON gi.item_id = gr.item_id
JOIN subjects sub ON sub.subject_id = gi.subject_id
JOIN semesters sem ON sem.semester_id = gi.semester_id
JOIN users u ON u.login = 'teacher_test'
WHERE sem.academic_year = '2026/2027'
  AND sem.term_num = 1
  AND sub.subject_index IN ('DEMO-ALG', 'DEMO-NET', 'DEMO-DB')
  AND gi.title LIKE '[DEMO]%'
  AND NOT EXISTS (
      SELECT 1 FROM grade_events existing
      WHERE existing.grade_id = gr.grade_id
        AND existing.event_type = 'grade_created'
  );

ALTER TABLE grades ENABLE TRIGGER trg_limit_grade_score;
ALTER TABLE grades ENABLE TRIGGER trg_grades_open_semester;
ALTER TABLE grade_items ENABLE TRIGGER trg_limit_subject_scores;
ALTER TABLE grade_items ENABLE TRIGGER trg_grade_items_open_semester;
-- +goose StatementEnd

-- +goose StatementBegin
-- Notifications for the profile feed and a system/fraud message for the dean.
DO $$
DECLARE
    v_teacher_user_id INTEGER;
    v_student_user_id INTEGER;
    v_dean_user_id INTEGER;
    v_notification_id BIGINT;
BEGIN
    SELECT id INTO v_teacher_user_id FROM users WHERE login = 'teacher_test';
    SELECT id INTO v_student_user_id FROM users WHERE login = 'student_test';
    SELECT id INTO v_dean_user_id FROM users WHERE login = 'dean_test';

    DELETE FROM notifications
    WHERE metadata ->> 'demo_key' = 'presentation-2026-2027';

    INSERT INTO notifications (
        category, event_type, title, message, created_by_user_id, metadata, expires_at, created_at
    )
    VALUES (
        'grades',
        'grade_created',
        'Выставлены новые оценки',
        'Преподаватель заполнил результаты по лабораторным и практическим работам.',
        v_teacher_user_id,
        '{"demo_key":"presentation-2026-2027","screen":"grades"}'::JSONB,
        TIMESTAMPTZ '2027-01-31 23:59:59+07',
        TIMESTAMPTZ '2026-08-25 14:10:00+07'
    )
    RETURNING notification_id INTO v_notification_id;
    INSERT INTO notification_recipients (notification_id, user_id)
    VALUES (v_notification_id, v_student_user_id);

    INSERT INTO notifications (
        category, event_type, title, message, created_by_user_id, metadata, expires_at, created_at
    )
    VALUES (
        'attendance',
        'attendance_marked',
        'Посещаемость обновлена',
        'Ручная отметка преподавателя учтена в текущем проценте посещаемости.',
        v_teacher_user_id,
        '{"demo_key":"presentation-2026-2027","screen":"attendance"}'::JSONB,
        TIMESTAMPTZ '2027-01-31 23:59:59+07',
        TIMESTAMPTZ '2026-08-26 09:45:00+07'
    )
    RETURNING notification_id INTO v_notification_id;
    INSERT INTO notification_recipients (notification_id, user_id)
    VALUES (v_notification_id, v_student_user_id);

    INSERT INTO notifications (
        category, event_type, title, message, created_by_user_id, metadata, expires_at, created_at
    )
    VALUES (
        'schedule',
        'lesson_rescheduled',
        'Расписание на неделю готово',
        'Для тестовых групп доступны занятия по двум типам недель.',
        v_teacher_user_id,
        '{"demo_key":"presentation-2026-2027","screen":"schedule"}'::JSONB,
        TIMESTAMPTZ '2027-01-31 23:59:59+07',
        TIMESTAMPTZ '2026-08-24 12:00:00+07'
    )
    RETURNING notification_id INTO v_notification_id;
    INSERT INTO notification_recipients (notification_id, user_id, read_at)
    VALUES
        (v_notification_id, v_student_user_id, TIMESTAMPTZ '2026-08-24 13:00:00+07'),
        (v_notification_id, v_teacher_user_id, NULL);

    INSERT INTO notifications (
        category, event_type, title, message, created_by_user_id, metadata, expires_at, created_at
    )
    VALUES (
        'system',
        'fraud',
        'Обнаружена подозрительная отметка',
        'Зафиксирована повторная отметка с устройства другого студента в группе DEMO-101.',
        v_teacher_user_id,
        '{"demo_key":"presentation-2026-2027","screen":"antifraud","severity":"high"}'::JSONB,
        TIMESTAMPTZ '2027-01-31 23:59:59+07',
        TIMESTAMPTZ '2026-08-18 08:14:00+07'
    )
    RETURNING notification_id INTO v_notification_id;
    INSERT INTO notification_recipients (notification_id, user_id)
    VALUES (v_notification_id, v_dean_user_id);
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE grade_items DISABLE TRIGGER trg_grade_items_open_semester;
ALTER TABLE grades DISABLE TRIGGER trg_grades_open_semester;
ALTER TABLE attendance_sessions DISABLE TRIGGER trg_attendance_sessions_open_semester;
ALTER TABLE attendance_session_students DISABLE TRIGGER trg_attendance_students_open_semester;
ALTER TABLE semesters DISABLE TRIGGER trg_semesters_validate_range;

DELETE FROM notifications
WHERE metadata ->> 'demo_key' = 'presentation-2026-2027';

DELETE FROM user_agreement_decisions
WHERE user_agent = 'presentation-demo-migration';

DELETE FROM grade_events ge
USING grade_items gi, semesters sem, subjects sub
WHERE ge.grade_item_id = gi.item_id
  AND gi.semester_id = sem.semester_id
  AND gi.subject_id = sub.subject_id
  AND sem.academic_year = '2026/2027'
  AND sem.term_num = 1
  AND sub.subject_index IN ('DEMO-ALG', 'DEMO-NET', 'DEMO-DB')
  AND gi.title LIKE '[DEMO]%';

DELETE FROM grades gr
USING grade_items gi, semesters sem, subjects sub
WHERE gr.item_id = gi.item_id
  AND gi.semester_id = sem.semester_id
  AND gi.subject_id = sub.subject_id
  AND sem.academic_year = '2026/2027'
  AND sem.term_num = 1
  AND sub.subject_index IN ('DEMO-ALG', 'DEMO-NET', 'DEMO-DB')
  AND gi.title LIKE '[DEMO]%';

DELETE FROM grade_items gi
USING semesters sem, subjects sub
WHERE gi.semester_id = sem.semester_id
  AND gi.subject_id = sub.subject_id
  AND sem.academic_year = '2026/2027'
  AND sem.term_num = 1
  AND sub.subject_index IN ('DEMO-ALG', 'DEMO-NET', 'DEMO-DB')
  AND gi.title LIKE '[DEMO]%';

DELETE FROM attendance_session_students ass
USING attendance_sessions sess, semesters sem
WHERE ass.session_id = sess.session_id
  AND sess.semester_id = sem.semester_id
  AND sem.academic_year = '2026/2027'
  AND sem.term_num = 1
  AND sess.lesson_name LIKE '[DEMO]%';

DELETE FROM attendance_session_groups asg
USING attendance_sessions sess, semesters sem
WHERE asg.session_id = sess.session_id
  AND sess.semester_id = sem.semester_id
  AND sem.academic_year = '2026/2027'
  AND sem.term_num = 1
  AND sess.lesson_name LIKE '[DEMO]%';

DELETE FROM attendance_sessions sess
USING semesters sem
WHERE sess.semester_id = sem.semester_id
  AND sem.academic_year = '2026/2027'
  AND sem.term_num = 1
  AND sess.lesson_name LIKE '[DEMO]%';

DELETE FROM schedules sch
USING semesters sem, subjects sub, groups g
WHERE sch.semester_id = sem.semester_id
  AND sch.subject_id = sub.subject_id
  AND sch.group_id = g.group_id
  AND sem.academic_year = '2026/2027'
  AND sem.term_num = 1
  AND sub.subject_index IN ('DEMO-ALG', 'DEMO-NET', 'DEMO-DB')
  AND g.group_name IN ('DEMO-101', 'DEMO-102');

DELETE FROM subject_controls sc
USING subjects sub, semesters sem
WHERE sc.subject_id = sub.subject_id
  AND sc.semester_id = sem.semester_id
  AND sub.subject_index IN ('DEMO-ALG', 'DEMO-NET', 'DEMO-DB')
  AND sem.academic_year = '2026/2027'
  AND sem.term_num = 1;

DELETE FROM semester_load sl
USING subjects sub, semesters sem
WHERE sl.subject_id = sub.subject_id
  AND sl.semester_id = sem.semester_id
  AND sub.subject_index IN ('DEMO-ALG', 'DEMO-NET', 'DEMO-DB')
  AND sem.academic_year = '2026/2027'
  AND sem.term_num = 1;

DELETE FROM subject_metrics sm
USING subjects sub
WHERE sm.subject_id = sub.subject_id
  AND sub.subject_index IN ('DEMO-ALG', 'DEMO-NET', 'DEMO-DB');

DELETE FROM subjects
WHERE subject_index IN ('DEMO-ALG', 'DEMO-NET', 'DEMO-DB');

UPDATE students
SET group_id = (
    SELECT group_id FROM groups WHERE group_name = 'TEST-GROUP-1' ORDER BY group_id LIMIT 1
)
WHERE user_id = (SELECT id FROM users WHERE login = 'student_test')
  AND EXISTS (SELECT 1 FROM groups WHERE group_name = 'TEST-GROUP-1');

DELETE FROM registration_invites
WHERE invite_code LIKE 'PRESENT-%';

DELETE FROM students
WHERE user_id IN (
    SELECT id FROM users
    WHERE login IN ('student_demo_2', 'student_demo_3', 'student_demo_4', 'student_demo_5')
);

DELETE FROM teachers
WHERE user_id = (SELECT id FROM users WHERE login = 'teacher_demo_2');

DELETE FROM users
WHERE login IN ('teacher_demo_2', 'student_demo_2', 'student_demo_3', 'student_demo_4', 'student_demo_5');

DELETE FROM groups
WHERE group_name IN ('DEMO-101', 'DEMO-102')
  AND NOT EXISTS (SELECT 1 FROM students WHERE students.group_id = groups.group_id);

DELETE FROM org_scopes
WHERE user_id = (SELECT id FROM users WHERE login = 'dean_test')
  AND faculty_id = (SELECT faculty_id FROM faculties WHERE code = 'DEMO-FIT');

INSERT INTO org_scopes (user_id, faculty_id)
SELECT u.id, f.faculty_id
FROM users u
CROSS JOIN faculties f
WHERE u.login = 'dean_test'
  AND f.code = 'FIT'
  AND NOT EXISTS (
      SELECT 1
      FROM org_scopes scope
      WHERE scope.user_id = u.id
        AND scope.faculty_id = f.faculty_id
  );

DELETE FROM lecterns
WHERE code = 'DEMO-IT'
  AND NOT EXISTS (SELECT 1 FROM groups WHERE groups.lectern_id = lecterns.lectern_id)
  AND NOT EXISTS (SELECT 1 FROM subjects WHERE subjects.lectern_id = lecterns.lectern_id);

DELETE FROM faculties
WHERE code = 'DEMO-FIT'
  AND NOT EXISTS (SELECT 1 FROM lecterns WHERE lecterns.faculty_id = faculties.faculty_id);

ALTER TABLE semesters DISABLE TRIGGER trg_semesters_validate_range;

DELETE FROM semesters
WHERE academic_year = '2026/2027'
  AND term_num = 1;

UPDATE semesters
SET ends_at = TIMESTAMPTZ '2027-06-30 23:59:59+07',
    status = 'open',
    is_current = TRUE,
    opened_at = COALESCE(opened_at, NOW()),
    closed_at = NULL,
    closed_by_user_id = NULL
WHERE academic_year = '2025/2026'
  AND term_num = 2;

ALTER TABLE semesters ENABLE TRIGGER trg_semesters_validate_range;
ALTER TABLE attendance_session_students ENABLE TRIGGER trg_attendance_students_open_semester;
ALTER TABLE attendance_sessions ENABLE TRIGGER trg_attendance_sessions_open_semester;
ALTER TABLE grades ENABLE TRIGGER trg_grades_open_semester;
ALTER TABLE grade_items ENABLE TRIGGER trg_grade_items_open_semester;

DO $$
DECLARE
    r RECORD;
    v_max BIGINT;
    v_seq TEXT;
BEGIN
    FOR r IN (
        SELECT table_name, column_name
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND (column_default LIKE 'nextval(%' OR is_identity = 'YES')
    ) LOOP
        v_seq := pg_get_serial_sequence(quote_ident(r.table_name), quote_ident(r.column_name));
        IF v_seq IS NOT NULL THEN
            EXECUTE format('SELECT COALESCE(MAX(%I), 0) FROM %I', r.column_name, r.table_name) INTO v_max;
            IF v_max > 0 THEN
                EXECUTE format('SELECT setval(%L, %s, true)', v_seq, v_max);
            END IF;
        END IF;
    END LOOP;
END $$;
-- +goose StatementEnd
