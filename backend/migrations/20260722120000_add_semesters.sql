-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS semesters (
    semester_id SERIAL PRIMARY KEY,
    academic_year VARCHAR(20) NOT NULL,
    term_num SMALLINT NOT NULL CHECK (term_num IN (1, 2)),
    name VARCHAR(255) NOT NULL,
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    is_current BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT semesters_year_term_unique UNIQUE (academic_year, term_num),
    CONSTRAINT semesters_range_check CHECK (ends_at > starts_at)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_semesters_current
    ON semesters (is_current)
    WHERE is_current;

ALTER TABLE attendance_sessions
    ADD COLUMN IF NOT EXISTS semester_id INT REFERENCES semesters(semester_id) ON DELETE RESTRICT;

ALTER TABLE grade_items
    ADD COLUMN IF NOT EXISTS semester_id INT REFERENCES semesters(semester_id) ON DELETE RESTRICT;

ALTER TABLE semester_load
    ADD COLUMN IF NOT EXISTS semester_id INT REFERENCES semesters(semester_id) ON DELETE RESTRICT;

ALTER TABLE subject_controls
    ADD COLUMN IF NOT EXISTS semester_id INT REFERENCES semesters(semester_id) ON DELETE RESTRICT;

ALTER TABLE attendance_sessions
    ADD COLUMN IF NOT EXISTS lesson_name VARCHAR(255),
    ADD COLUMN IF NOT EXISTS lat DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS lon DOUBLE PRECISION;

INSERT INTO semesters (academic_year, term_num, name, starts_at, ends_at, is_current)
VALUES
    ('2025/2026', 1, '2025/2026, осенний семестр', TIMESTAMPTZ '2025-09-01 00:00:00+07', TIMESTAMPTZ '2026-01-31 23:59:59+07', FALSE),
    ('2025/2026', 2, '2025/2026, весенний семестр', TIMESTAMPTZ '2026-02-01 00:00:00+07', TIMESTAMPTZ '2027-06-30 23:59:59+07', TRUE)
ON CONFLICT (academic_year, term_num) DO UPDATE
SET name = EXCLUDED.name,
    starts_at = EXCLUDED.starts_at,
    ends_at = EXCLUDED.ends_at,
    is_current = EXCLUDED.is_current;

UPDATE grade_items gi
SET semester_id = COALESCE(
    (
        SELECT s.semester_id
        FROM semesters s
        WHERE COALESCE(gi.deadline, gi.created_at) >= s.starts_at
          AND COALESCE(gi.deadline, gi.created_at) <= s.ends_at
        ORDER BY s.starts_at DESC
        LIMIT 1
    ),
    (
        SELECT semester_id
        FROM semesters
        WHERE is_current = TRUE
        ORDER BY semester_id DESC
        LIMIT 1
    )
)
WHERE gi.semester_id IS NULL;

UPDATE attendance_sessions s
SET semester_id = COALESCE(
    (
        SELECT sem.semester_id
        FROM semesters sem
        WHERE s.created_at >= sem.starts_at
          AND s.created_at <= sem.ends_at
        ORDER BY sem.starts_at DESC
        LIMIT 1
    ),
    (
        SELECT semester_id
        FROM semesters
        WHERE is_current = TRUE
        ORDER BY semester_id DESC
        LIMIT 1
    )
)
WHERE s.semester_id IS NULL;

UPDATE semester_load sl
SET semester_id = COALESCE(
    (
        SELECT semester_id
        FROM semesters
        WHERE academic_year = '2025/2026'
          AND term_num = sl.semester_num
        LIMIT 1
    ),
    (
        SELECT semester_id
        FROM semesters
        WHERE is_current = TRUE
        ORDER BY semester_id DESC
        LIMIT 1
    )
)
WHERE sl.semester_id IS NULL;

UPDATE subject_controls sc
SET semester_id = COALESCE(
    (
        SELECT semester_id
        FROM semesters
        WHERE academic_year = '2025/2026'
          AND term_num = sc.semester_num
        LIMIT 1
    ),
    (
        SELECT semester_id
        FROM semesters
        WHERE is_current = TRUE
        ORDER BY semester_id DESC
        LIMIT 1
    )
)
WHERE sc.semester_id IS NULL;

ALTER TABLE grade_items
    ALTER COLUMN semester_id SET NOT NULL;

ALTER TABLE attendance_sessions
    ALTER COLUMN semester_id SET NOT NULL;

ALTER TABLE semester_load
    ALTER COLUMN semester_id SET NOT NULL;

ALTER TABLE subject_controls
    ALTER COLUMN semester_id SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_grade_items_subject_semester_active
    ON grade_items (subject_id, semester_id, deadline, item_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_attendance_sessions_semester_teacher
    ON attendance_sessions (semester_id, teacher_id, expires_at DESC);

CREATE INDEX IF NOT EXISTS idx_semester_load_subject_semester
    ON semester_load (subject_id, semester_id);

CREATE INDEX IF NOT EXISTS idx_subject_controls_subject_semester
    ON subject_controls (subject_id, semester_id);

CREATE OR REPLACE FUNCTION public.check_subject_score_limit()
RETURNS TRIGGER AS $$
DECLARE
    total_max INTEGER;
BEGIN
    SELECT COALESCE(SUM(max_score), 0)
    INTO total_max
    FROM public.grade_items
    WHERE subject_id = NEW.subject_id
      AND semester_id = NEW.semester_id
      AND deleted_at IS NULL
      AND item_id IS DISTINCT FROM NEW.item_id;

    IF NEW.deleted_at IS NULL AND (total_max + NEW.max_score) > 100 THEN
        RAISE EXCEPTION 'Суммарный балл по предмету (БРС) не может превышать 100. Текущая сумма: %', total_max;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_subject_controls_subject_semester;
DROP INDEX IF EXISTS idx_semester_load_subject_semester;
DROP INDEX IF EXISTS idx_attendance_sessions_semester_teacher;
DROP INDEX IF EXISTS idx_grade_items_subject_semester_active;

ALTER TABLE subject_controls DROP COLUMN IF EXISTS semester_id;
ALTER TABLE semester_load DROP COLUMN IF EXISTS semester_id;
ALTER TABLE grade_items DROP COLUMN IF EXISTS semester_id;
ALTER TABLE attendance_sessions DROP COLUMN IF EXISTS semester_id;

DROP TABLE IF EXISTS semesters;

CREATE OR REPLACE FUNCTION public.check_subject_score_limit()
RETURNS TRIGGER AS $$
DECLARE
    total_max INTEGER;
BEGIN
    SELECT COALESCE(SUM(max_score), 0)
    INTO total_max
    FROM public.grade_items
    WHERE subject_id = NEW.subject_id
      AND deleted_at IS NULL
      AND item_id IS DISTINCT FROM NEW.item_id;

    IF NEW.deleted_at IS NULL AND (total_max + NEW.max_score) > 100 THEN
        RAISE EXCEPTION 'Суммарный балл по предмету (БРС) не может превышать 100. Текущая сумма: %', total_max;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd
