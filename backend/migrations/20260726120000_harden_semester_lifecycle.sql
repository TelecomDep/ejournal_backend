-- +goose Up
-- +goose StatementBegin
ALTER TABLE semesters
    ADD COLUMN status VARCHAR(20),
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ADD COLUMN created_by_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN opened_at TIMESTAMPTZ,
    ADD COLUMN opened_by_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN closed_at TIMESTAMPTZ,
    ADD COLUMN closed_by_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN archived_at TIMESTAMPTZ,
    ADD COLUMN archived_by_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL;

UPDATE semesters
SET status = CASE
        WHEN is_current AND starts_at <= now() AND ends_at > now() THEN 'open'
        WHEN ends_at <= now() THEN 'closed'
        ELSE 'planned'
    END,
    opened_at = CASE
        WHEN is_current AND starts_at <= now() AND ends_at > now() THEN COALESCE(created_at, now())
        ELSE NULL
    END;

UPDATE semesters
SET is_current = (status = 'open');

ALTER TABLE semesters
    ALTER COLUMN status SET DEFAULT 'planned',
    ALTER COLUMN status SET NOT NULL,
    ADD CONSTRAINT semesters_status_check
        CHECK (status IN ('planned', 'open', 'closed', 'archived')),
    ADD CONSTRAINT semesters_open_current_check
        CHECK (is_current = (status = 'open')),
    ADD CONSTRAINT semesters_academic_year_format_check
        CHECK (academic_year ~ '^[0-9]{4}/[0-9]{4}$');

CREATE UNIQUE INDEX idx_semesters_single_open
    ON semesters (status)
    WHERE status = 'open';

CREATE OR REPLACE FUNCTION check_semester_date_overlap()
RETURNS TRIGGER AS $$
BEGIN
    PERFORM pg_advisory_xact_lock(7365636573746572);

    IF EXISTS (
        SELECT 1
        FROM semesters s
        WHERE s.semester_id IS DISTINCT FROM NEW.semester_id
          AND NEW.starts_at < s.ends_at
          AND NEW.ends_at > s.starts_at
    ) THEN
        RAISE EXCEPTION 'semester date range overlaps an existing semester'
            USING ERRCODE = '23P01';
    END IF;

    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_semesters_validate_range
BEFORE INSERT OR UPDATE OF starts_at, ends_at ON semesters
FOR EACH ROW
EXECUTE FUNCTION check_semester_date_overlap();

-- The legacy semester_num columns represent a curriculum term (1..8), not an
-- academic calendar period. Keep them authoritative and make the calendar link
-- optional until a curriculum/cohort mapping exists.
ALTER TABLE semester_load
    ALTER COLUMN semester_id DROP NOT NULL;

ALTER TABLE subject_controls
    ALTER COLUMN semester_id DROP NOT NULL;

UPDATE semester_load SET semester_id = NULL;
UPDATE subject_controls SET semester_id = NULL;

COMMENT ON COLUMN semester_load.semester_num IS
    'Curriculum semester number (usually 1..8), not semesters.term_num';
COMMENT ON COLUMN subject_controls.semester_num IS
    'Curriculum semester number (usually 1..8), not semesters.term_num';

ALTER TABLE schedules
    ADD COLUMN semester_id INTEGER REFERENCES semesters(semester_id) ON DELETE RESTRICT;

UPDATE schedules
SET semester_id = (
    SELECT semester_id
    FROM semesters
    ORDER BY
        CASE WHEN starts_at <= now() AND ends_at > now() THEN 0 ELSE 1 END,
        starts_at DESC,
        semester_id DESC
    LIMIT 1
)
WHERE semester_id IS NULL;

ALTER TABLE schedules
    ALTER COLUMN semester_id SET NOT NULL;

CREATE INDEX idx_schedules_semester_teacher
    ON schedules (semester_id, teacher_id, day_idx, lesson_num);

CREATE INDEX idx_schedules_semester_group
    ON schedules (semester_id, group_id, day_idx, lesson_num);

CREATE OR REPLACE FUNCTION assign_schedule_semester()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.semester_id IS NULL THEN
        SELECT semester_id
        INTO NEW.semester_id
        FROM semesters
        WHERE status = 'open'
        ORDER BY semester_id DESC
        LIMIT 1;
    END IF;

    IF NEW.semester_id IS NULL THEN
        RAISE EXCEPTION 'an open semester is required before importing a schedule'
            USING ERRCODE = '23502';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_schedules_assign_semester
BEFORE INSERT ON schedules
FOR EACH ROW
EXECUTE FUNCTION assign_schedule_semester();

CREATE OR REPLACE FUNCTION enforce_open_semester_write()
RETURNS TRIGGER AS $$
DECLARE
    target_semester_id INTEGER;
BEGIN
    CASE TG_TABLE_NAME
        WHEN 'grade_items' THEN
            target_semester_id := CASE WHEN TG_OP = 'DELETE' THEN OLD.semester_id ELSE NEW.semester_id END;
        WHEN 'attendance_sessions' THEN
            target_semester_id := CASE WHEN TG_OP = 'DELETE' THEN OLD.semester_id ELSE NEW.semester_id END;
        WHEN 'grades' THEN
            SELECT gi.semester_id
            INTO target_semester_id
            FROM grade_items gi
            WHERE gi.item_id = CASE WHEN TG_OP = 'DELETE' THEN OLD.item_id ELSE NEW.item_id END;
        WHEN 'attendance_session_students' THEN
            SELECT s.semester_id
            INTO target_semester_id
            FROM attendance_sessions s
            WHERE s.session_id = CASE WHEN TG_OP = 'DELETE' THEN OLD.session_id ELSE NEW.session_id END;
        ELSE
            RAISE EXCEPTION 'unsupported semester-protected table: %', TG_TABLE_NAME;
    END CASE;

    PERFORM 1
    FROM semesters sem
    WHERE sem.semester_id = target_semester_id
      AND sem.status = 'open'
      AND sem.is_current = TRUE
      AND now() >= sem.starts_at
      AND now() < sem.ends_at
    FOR SHARE;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'semester is not open for changes'
            USING ERRCODE = '55000';
    END IF;

    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_grade_items_open_semester
BEFORE INSERT OR UPDATE OR DELETE ON grade_items
FOR EACH ROW
EXECUTE FUNCTION enforce_open_semester_write();

CREATE TRIGGER trg_grades_open_semester
BEFORE INSERT OR UPDATE OR DELETE ON grades
FOR EACH ROW
EXECUTE FUNCTION enforce_open_semester_write();

CREATE TRIGGER trg_attendance_sessions_open_semester
BEFORE INSERT OR UPDATE OR DELETE ON attendance_sessions
FOR EACH ROW
EXECUTE FUNCTION enforce_open_semester_write();

CREATE TRIGGER trg_attendance_students_open_semester
BEFORE INSERT OR UPDATE OR DELETE ON attendance_session_students
FOR EACH ROW
EXECUTE FUNCTION enforce_open_semester_write();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS trg_attendance_students_open_semester ON attendance_session_students;
DROP TRIGGER IF EXISTS trg_attendance_sessions_open_semester ON attendance_sessions;
DROP TRIGGER IF EXISTS trg_grades_open_semester ON grades;
DROP TRIGGER IF EXISTS trg_grade_items_open_semester ON grade_items;
DROP FUNCTION IF EXISTS enforce_open_semester_write();

DROP TRIGGER IF EXISTS trg_schedules_assign_semester ON schedules;
DROP FUNCTION IF EXISTS assign_schedule_semester();
DROP INDEX IF EXISTS idx_schedules_semester_group;
DROP INDEX IF EXISTS idx_schedules_semester_teacher;
ALTER TABLE schedules DROP COLUMN IF EXISTS semester_id;

UPDATE semester_load
SET semester_id = (
    SELECT semester_id
    FROM semesters
    ORDER BY is_current DESC, semester_id DESC
    LIMIT 1
)
WHERE semester_id IS NULL;

UPDATE subject_controls
SET semester_id = (
    SELECT semester_id
    FROM semesters
    ORDER BY is_current DESC, semester_id DESC
    LIMIT 1
)
WHERE semester_id IS NULL;

ALTER TABLE semester_load
    ALTER COLUMN semester_id SET NOT NULL;

ALTER TABLE subject_controls
    ALTER COLUMN semester_id SET NOT NULL;

DROP TRIGGER IF EXISTS trg_semesters_validate_range ON semesters;
DROP FUNCTION IF EXISTS check_semester_date_overlap();
DROP INDEX IF EXISTS idx_semesters_single_open;

ALTER TABLE semesters
    DROP CONSTRAINT IF EXISTS semesters_academic_year_format_check,
    DROP CONSTRAINT IF EXISTS semesters_open_current_check,
    DROP CONSTRAINT IF EXISTS semesters_status_check,
    DROP COLUMN IF EXISTS archived_by_user_id,
    DROP COLUMN IF EXISTS archived_at,
    DROP COLUMN IF EXISTS closed_by_user_id,
    DROP COLUMN IF EXISTS closed_at,
    DROP COLUMN IF EXISTS opened_by_user_id,
    DROP COLUMN IF EXISTS opened_at,
    DROP COLUMN IF EXISTS created_by_user_id,
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS status;
-- +goose StatementEnd
