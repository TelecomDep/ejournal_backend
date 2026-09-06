package app

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/TelecomDep/ejournal_backend/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestCreateAttendanceLinkByTeacherAuthorization exercises the real teacher
// attendance creation path against an isolated schema. It is intentionally
// opt-in because the repository does not own a PostgreSQL test server.
//
// Run with, for example:
//
//	TEST_DB_DSN='postgres://postgres:postgres@127.0.0.1:5432/ejournal_attendance_test?sslmode=disable' \
//	  go test ./internal/app -run TestCreateAttendanceLinkByTeacherAuthorization -count=1
func TestCreateAttendanceLinkByTeacherAuthorization(t *testing.T) {
	baseDSN := strings.TrimSpace(os.Getenv("TEST_DB_DSN"))
	if baseDSN == "" {
		t.Skip("TEST_DB_DSN is not set; skipping PostgreSQL authorization integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	adminPool, err := pgxpool.New(ctx, baseDSN)
	if err != nil {
		t.Fatalf("connect to TEST_DB_DSN: %v", err)
	}
	defer adminPool.Close()
	if err := adminPool.Ping(ctx); err != nil {
		t.Fatalf("ping TEST_DB_DSN: %v", err)
	}

	var databaseName string
	if err := adminPool.QueryRow(ctx, "SELECT current_database()").Scan(&databaseName); err != nil {
		t.Fatalf("identify TEST_DB_DSN database: %v", err)
	}
	if strings.EqualFold(strings.TrimSpace(databaseName), "ejournal") {
		t.Fatalf("refusing to run authorization integration test against application database %q; use a dedicated TEST_DB_DSN", databaseName)
	}

	schema := fmt.Sprintf("attendance_auth_test_%d", time.Now().UnixNano())
	if _, err := adminPool.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatalf("create isolated schema %q: %v", schema, err)
	}
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if _, err := adminPool.Exec(cleanupCtx, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Errorf("drop isolated schema %q: %v", schema, err)
		}
	}()

	if _, err := adminPool.Exec(ctx, attendanceAuthorizationSchemaSQL(schema)); err != nil {
		t.Fatalf("create isolated authorization fixture: %v", err)
	}
	if _, err := adminPool.Exec(ctx, attendanceAuthorizationFixtureSQL(schema)); err != nil {
		t.Fatalf("seed isolated authorization fixture: %v", err)
	}

	storeDSN, err := attendanceAuthorizationDSN(baseDSN, schema)
	if err != nil {
		t.Fatalf("configure isolated schema search_path: %v", err)
	}
	store, err := db.NewStore(ctx, storeDSN)
	if err != nil {
		t.Fatalf("create isolated store: %v", err)
	}
	defer store.Close()
	if err := store.Ping(ctx); err != nil {
		t.Fatalf("ping isolated store: %v", err)
	}

	service := NewService(
		"test-secret-with-sufficient-length",
		"https://attendance.example.test",
		"",
		"",
		0,
		true,
		store,
		nil,
	)
	token, err := service.generateJWTForRole(101, 1, RoleTeacher)
	if err != nil {
		t.Fatalf("generate teacher session token: %v", err)
	}

	semesterID := int32(30)
	tests := []struct {
		name          string
		subjectID     int32
		groupIDs      []int32
		wantOK        bool
		wantError     string
		wantNewRecord bool
	}{
		{
			name:          "elective assigned subject and group is allowed",
			subjectID:     10,
			groupIDs:      []int32{20},
			wantOK:        true,
			wantNewRecord: true,
		},
		{
			name:      "elective foreign subject is rejected",
			subjectID: 11,
			groupIDs:  []int32{20},
			wantError: "forbidden: teacher is not assigned to subject",
		},
		{
			name:      "elective foreign group is rejected",
			subjectID: 10,
			groupIDs:  []int32{21},
			wantError: "forbidden: group is not assigned to teacher for this subject",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			before := attendanceAuthorizationSessionCount(t, store.Pool())
			response := service.createAttendanceLinkByTeacher(token, AttendanceCreateData{
				SubjectID:      test.subjectID,
				SemesterID:     &semesterID,
				GroupIDs:       test.groupIDs,
				LessonType:     "факультатив",
				ExpiresMinutes: 5,
			})

			if response.OK != test.wantOK {
				t.Fatalf("createAttendanceLinkByTeacher().OK = %v, want %v (error %q)", response.OK, test.wantOK, response.Error)
			}
			if !test.wantOK && response.Error != test.wantError {
				t.Fatalf("createAttendanceLinkByTeacher().Error = %q, want %q", response.Error, test.wantError)
			}

			after := attendanceAuthorizationSessionCount(t, store.Pool())
			wantAfter := before
			if test.wantNewRecord {
				wantAfter++
			}
			if after != wantAfter {
				t.Fatalf("attendance_sessions count = %d, want %d", after, wantAfter)
			}
		})
	}
}

func attendanceAuthorizationSessionCount(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var count int
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM attendance_sessions").Scan(&count); err != nil {
		t.Fatalf("count attendance sessions: %v", err)
	}
	return count
}

func attendanceAuthorizationDSN(baseDSN, schema string) (string, error) {
	parsed, err := url.Parse(baseDSN)
	if err == nil && parsed.Scheme != "" && parsed.Host != "" {
		query := parsed.Query()
		query.Set("options", "-c search_path="+schema+",public")
		parsed.RawQuery = query.Encode()
		return parsed.String(), nil
	}

	// pgx also accepts libpq keyword/value DSNs. The generated schema contains
	// only safe identifier characters, so it is safe to quote this option.
	return strings.TrimSpace(baseDSN) + " options='-c search_path=" + schema + ",public'", nil
}

func attendanceAuthorizationSchemaSQL(schema string) string {
	return fmt.Sprintf(`
CREATE TABLE %s.users (
    id INTEGER PRIMARY KEY,
    login TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL,
    email TEXT,
    status TEXT NOT NULL DEFAULT 'active',
    is_2fa_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    token_version BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE %s.user_roles (
    user_id INTEGER NOT NULL REFERENCES %s.users(id) ON DELETE CASCADE,
    role TEXT NOT NULL,
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (user_id, role)
);

CREATE TABLE %s.subjects (
    subject_id INTEGER PRIMARY KEY,
    subject_index TEXT,
    name TEXT NOT NULL,
    in_plan BOOLEAN NOT NULL DEFAULT TRUE,
    lectern_id INTEGER
);

CREATE TABLE %s.groups (
    group_id INTEGER PRIMARY KEY,
    group_name TEXT NOT NULL,
    lectern_id INTEGER
);

CREATE TABLE %s.teachers (
    teacher_id INTEGER PRIMARY KEY,
    role TEXT NOT NULL DEFAULT 'teacher',
    user_id INTEGER REFERENCES %s.users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    lectern_id INTEGER,
    job_title TEXT
);

CREATE TABLE %s.students (
    student_id INTEGER PRIMARY KEY,
    user_id INTEGER REFERENCES %s.users(id) ON DELETE CASCADE,
    student_name TEXT,
    group_id INTEGER REFERENCES %s.groups(group_id) ON DELETE CASCADE
);

CREATE TABLE %s.semesters (
    semester_id INTEGER PRIMARY KEY,
    academic_year TEXT NOT NULL,
    term_num INTEGER NOT NULL,
    name TEXT NOT NULL,
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL,
    is_current BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by_user_id INTEGER,
    opened_at TIMESTAMPTZ,
    opened_by_user_id INTEGER,
    closed_at TIMESTAMPTZ,
    closed_by_user_id INTEGER,
    archived_at TIMESTAMPTZ,
    archived_by_user_id INTEGER
);

CREATE TABLE %s.schedules (
    schedule_id SERIAL PRIMARY KEY,
    group_id INTEGER NOT NULL REFERENCES %s.groups(group_id) ON DELETE CASCADE,
    subject_id INTEGER NOT NULL REFERENCES %s.subjects(subject_id) ON DELETE CASCADE,
    teacher_id INTEGER NOT NULL REFERENCES %s.teachers(teacher_id) ON DELETE CASCADE,
    semester_id INTEGER NOT NULL REFERENCES %s.semesters(semester_id) ON DELETE CASCADE
);

CREATE TABLE %s.attendance_sessions (
    session_id SERIAL PRIMARY KEY,
    teacher_id INTEGER NOT NULL REFERENCES %s.teachers(teacher_id) ON DELETE CASCADE,
    subject_id INTEGER NOT NULL REFERENCES %s.subjects(subject_id) ON DELETE CASCADE,
    semester_id INTEGER NOT NULL REFERENCES %s.semesters(semester_id) ON DELETE RESTRICT,
    lesson_name TEXT,
    lat DOUBLE PRECISION,
    lon DOUBLE PRECISION,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE %s.attendance_session_groups (
    session_id INTEGER NOT NULL REFERENCES %s.attendance_sessions(session_id) ON DELETE CASCADE,
    group_id INTEGER NOT NULL REFERENCES %s.groups(group_id) ON DELETE CASCADE,
    PRIMARY KEY (session_id, group_id)
);

CREATE TABLE %s.attendance_session_students (
    session_id INTEGER NOT NULL REFERENCES %s.attendance_sessions(session_id) ON DELETE CASCADE,
    student_id INTEGER NOT NULL REFERENCES %s.students(student_id) ON DELETE CASCADE,
    group_id_snapshot INTEGER NOT NULL REFERENCES %s.groups(group_id) ON DELETE RESTRICT,
    status TEXT NOT NULL DEFAULT 'absent',
    marked_at TIMESTAMPTZ,
    marked_by TEXT,
    PRIMARY KEY (session_id, student_id)
);

CREATE TABLE %s.notifications (
    notification_id BIGSERIAL PRIMARY KEY,
    category TEXT NOT NULL,
    event_type TEXT NOT NULL,
    title TEXT NOT NULL,
    message TEXT NOT NULL,
    created_by_user_id INTEGER REFERENCES %s.users(id) ON DELETE SET NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE %s.notification_recipients (
    notification_id BIGINT NOT NULL REFERENCES %s.notifications(notification_id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES %s.users(id) ON DELETE CASCADE,
    read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (notification_id, user_id)
);
`,
		schema,
		schema, schema,
		schema,
		schema,
		schema, schema,
		schema, schema, schema,
		schema,
		schema, schema, schema, schema, schema,
		schema, schema, schema, schema,
		schema, schema, schema,
		schema, schema, schema, schema,
		schema, schema,
		schema, schema, schema,
	)
}

func attendanceAuthorizationFixtureSQL(schema string) string {
	return fmt.Sprintf(`
INSERT INTO %s.users (id, login, password_hash, role, status, token_version)
VALUES (101, 'attendance_auth_teacher', 'unused', 'teacher', 'active', 1);

INSERT INTO %s.user_roles (user_id, role, is_primary)
VALUES (101, 'teacher', TRUE);

INSERT INTO %s.teachers (teacher_id, user_id, name)
VALUES (201, 101, 'Authorization Test Teacher');

INSERT INTO %s.subjects (subject_id, subject_index, name)
VALUES (10, 'ASSIGNED', 'Assigned Subject'),
       (11, 'FOREIGN', 'Foreign Subject');

INSERT INTO %s.groups (group_id, group_name)
VALUES (20, 'Assigned Group'),
       (21, 'Foreign Group');

INSERT INTO %s.semesters (
    semester_id, academic_year, term_num, name, starts_at, ends_at, status, is_current
)
VALUES (
    30, '2099/2100', 1, 'Authorization Test Semester',
    NOW() - INTERVAL '1 day', NOW() + INTERVAL '1 day', 'open', TRUE
);

INSERT INTO %s.schedules (group_id, subject_id, teacher_id, semester_id)
VALUES (20, 10, 201, 30);
`, schema, schema, schema, schema, schema, schema, schema)
}
