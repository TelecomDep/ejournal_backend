-- +goose Up
-- +goose StatementBegin

-- 0. Удаляем зависимые вьюхи, которые блокируют изменение колонок
DROP VIEW IF EXISTS view_attendance_journal;
DROP VIEW IF EXISTS view_next_lessons;

-- 1. Обновление таблицы lecterns
ALTER TABLE lecterns ALTER COLUMN code TYPE VARCHAR(20);
ALTER TABLE lecterns ADD CONSTRAINT lecterns_code_unique UNIQUE (code);

-- 2. Обновление таблицы groups
ALTER TABLE groups ALTER COLUMN group_name TYPE VARCHAR(50);
ALTER TABLE groups ADD CONSTRAINT groups_group_name_unique UNIQUE (group_name);

-- 3. Обновление таблицы teachers
ALTER TABLE teachers ALTER COLUMN name TYPE VARCHAR(255);
ALTER TABLE teachers ALTER COLUMN name SET NOT NULL;
ALTER TABLE teachers ALTER COLUMN job_title TYPE VARCHAR(255);

-- 4. Обновление таблицы students
ALTER TABLE students ALTER COLUMN student_name TYPE VARCHAR(255);
ALTER TABLE students ALTER COLUMN student_name SET NOT NULL;
ALTER TABLE students ADD CONSTRAINT unique_student_group UNIQUE (student_name, group_id);

-- 5. Пересоздаем вьюхи (уже с новыми типами колонок)

-- Журнал для преподавателя
CREATE OR REPLACE VIEW view_attendance_journal AS
SELECT 
    g.group_name,
    st.student_name,
    st.nfc_id,
    sub.name AS subject_name,
    sch.lesson_num,
    COALESCE(att.status, FALSE) AS is_present,
    att.check_in_time,
    CURRENT_DATE as report_date
FROM students st
JOIN groups g ON st.group_id = g.group_id
JOIN schedules sch ON g.group_id = sch.group_id
JOIN subjects sub ON sch.subject_id = sub.subject_id
LEFT JOIN attendance att ON st.student_id = att.student_id 
    AND sch.schedule_id = att.schedule_id 
    AND att.lesson_date = CURRENT_DATE;

-- Ближайшие занятия
CREATE OR REPLACE VIEW view_next_lessons AS
WITH current_data AS (
    SELECT 
        g.group_id,
        g.group_name,
        s.schedule_id,
        sub.name AS subject_name,
        t.name AS teacher_name,
        s.lesson_type,
        s.room_info,
        lt.start_time,
        s.day_idx, 
        CASE WHEN extract(dow from now()) = 0 THEN 7 ELSE extract(dow from now()) END as today_idx
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
WHERE today_idx = day_idx
  AND start_time > current_time
ORDER BY group_id, start_time ASC;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- В откате миграции просто удаляем констрейнты
ALTER TABLE students DROP CONSTRAINT IF EXISTS unique_student_group;
ALTER TABLE groups DROP CONSTRAINT IF EXISTS groups_group_name_unique;
ALTER TABLE lecterns DROP CONSTRAINT IF EXISTS lecterns_code_unique;
-- +goose StatementEnd
