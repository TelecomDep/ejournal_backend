-- +goose Up
-- +goose StatementBegin
INSERT INTO control_types (type_name)
SELECT unnest(ARRAY['Экзамен', 'Зачет', 'Зачет с оц.', 'КП', 'КР', 'Реферат', 'РГР'])
ON CONFLICT (type_name) DO NOTHING;

ALTER TABLE teachers
    ADD COLUMN IF NOT EXISTS user_id INTEGER REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE students
    ADD COLUMN IF NOT EXISTS user_id INTEGER REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE students
    ADD COLUMN IF NOT EXISTS nfc_id VARCHAR(100);

UPDATE teachers
SET user_id = teacher_id
WHERE user_id IS NULL;

UPDATE students
SET user_id = student_id
WHERE user_id IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_students_nfc_id_unique
    ON students (nfc_id)
    WHERE nfc_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS lesson_times (
    lesson_num INTEGER PRIMARY KEY,
    start_time TIME NOT NULL,
    end_time TIME NOT NULL
);

INSERT INTO lesson_times (lesson_num, start_time, end_time)
VALUES
    (1, '08:00', '09:35'),
    (2, '09:50', '11:25'),
    (3, '11:40', '13:15'),
    (4, '14:00', '15:35'),
    (5, '16:10', '17:45'),
    (6, '18:00', '19:35')
ON CONFLICT (lesson_num) DO NOTHING;

CREATE TABLE IF NOT EXISTS schedules (
    schedule_id SERIAL PRIMARY KEY,
    group_id INTEGER REFERENCES groups(group_id) ON DELETE CASCADE,
    subject_id INTEGER REFERENCES subjects(subject_id) ON DELETE CASCADE,
    teacher_id INTEGER REFERENCES teachers(teacher_id) ON DELETE CASCADE,
    lesson_num INTEGER REFERENCES lesson_times(lesson_num),
    day_idx INTEGER NOT NULL CHECK (day_idx BETWEEN 1 AND 7),
    subgroup VARCHAR(255),
    lesson_type VARCHAR(255),
    room_info VARCHAR(500)
);

CREATE INDEX IF NOT EXISTS idx_schedules_group_day_lesson
    ON schedules (group_id, day_idx, lesson_num);

CREATE INDEX IF NOT EXISTS idx_schedules_teacher_day_lesson
    ON schedules (teacher_id, day_idx, lesson_num);

CREATE TABLE IF NOT EXISTS attendance (
    attendance_id SERIAL PRIMARY KEY,
    student_id INTEGER NOT NULL REFERENCES students(student_id) ON DELETE CASCADE,
    schedule_id INTEGER NOT NULL REFERENCES schedules(schedule_id) ON DELETE CASCADE,
    lesson_date DATE NOT NULL DEFAULT CURRENT_DATE,
    status BOOLEAN DEFAULT FALSE,
    check_in_time TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (student_id, schedule_id, lesson_date)
);

CREATE INDEX IF NOT EXISTS idx_attendance_schedule_date
    ON attendance (schedule_id, lesson_date);

CREATE INDEX IF NOT EXISTS idx_attendance_student_date
    ON attendance (student_id, lesson_date);

CREATE OR REPLACE VIEW view_next_lessons AS
WITH current_data AS (
    SELECT
        g.group_id,
        g.group_name,
        s.schedule_id,
        s.day_idx,
        sub.name AS subject_name,
        t.name AS teacher_name,
        s.lesson_type,
        s.room_info,
        lt.start_time,
        CASE
            WHEN EXTRACT(DOW FROM NOW()) = 0 THEN 7
            ELSE EXTRACT(DOW FROM NOW())
        END::INTEGER AS today_idx
    FROM groups g
    JOIN schedules s ON g.group_id = s.group_id
    JOIN subjects sub ON s.subject_id = sub.subject_id
    JOIN teachers t ON s.teacher_id = t.teacher_id
    JOIN lesson_times lt ON s.lesson_num = lt.lesson_num
)
SELECT DISTINCT ON (group_id)
    group_id,
    group_name,
    subject_name,
    teacher_name,
    lesson_type,
    room_info,
    start_time
FROM current_data
WHERE day_idx = today_idx
  AND start_time > CURRENT_TIME
ORDER BY group_id, start_time ASC;

CREATE OR REPLACE VIEW view_attendance_journal AS
SELECT
    g.group_name,
    st.student_name,
    st.nfc_id,
    sub.name AS subject_name,
    sch.lesson_num,
    COALESCE(att.status, FALSE) AS is_present,
    att.check_in_time,
    CURRENT_DATE AS report_date
FROM students st
JOIN groups g ON st.group_id = g.group_id
JOIN schedules sch ON g.group_id = sch.group_id
JOIN subjects sub ON sch.subject_id = sub.subject_id
LEFT JOIN attendance att
       ON st.student_id = att.student_id
      AND sch.schedule_id = att.schedule_id
      AND att.lesson_date = CURRENT_DATE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP VIEW IF EXISTS view_attendance_journal;
DROP VIEW IF EXISTS view_next_lessons;

DROP INDEX IF EXISTS idx_attendance_student_date;
DROP INDEX IF EXISTS idx_attendance_schedule_date;
DROP TABLE IF EXISTS attendance;

DROP INDEX IF EXISTS idx_schedules_teacher_day_lesson;
DROP INDEX IF EXISTS idx_schedules_group_day_lesson;
DROP TABLE IF EXISTS schedules;
DROP TABLE IF EXISTS lesson_times;

DROP INDEX IF EXISTS idx_students_nfc_id_unique;

ALTER TABLE students DROP COLUMN IF EXISTS nfc_id;
ALTER TABLE students DROP COLUMN IF EXISTS user_id;
ALTER TABLE teachers DROP COLUMN IF EXISTS user_id;
-- +goose StatementEnd
