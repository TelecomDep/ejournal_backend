-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    CREATE TYPE user_role AS ENUM ('teacher', 'student', 'admin');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    login VARCHAR(255) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role user_role NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (id, role)
);

CREATE TABLE IF NOT EXISTS lecterns (
    lectern_id SERIAL PRIMARY KEY,
    code VARCHAR(10),
    name VARCHAR(255) NOT NULL
);

CREATE TABLE IF NOT EXISTS control_types (
    type_id SERIAL PRIMARY KEY,
    type_name VARCHAR(50) UNIQUE NOT NULL
);

CREATE TABLE IF NOT EXISTS subjects (
    subject_id SERIAL PRIMARY KEY,
    subject_index VARCHAR(20),
    name VARCHAR(255) NOT NULL,
    in_plan BOOLEAN DEFAULT TRUE,
    lectern_id INT REFERENCES lecterns(lectern_id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS subject_metrics (
    subject_id INT PRIMARY KEY REFERENCES subjects(subject_id) ON DELETE CASCADE,
    zet_expert INT,
    zet_fact INT,
    hours_expert INT,
    hours_by_plan INT,
    hours_contr_work INT,
    hours_auditory INT,
    hours_self_study INT,
    hours_control INT,
    hours_prep INT
);

CREATE TABLE IF NOT EXISTS semester_load (
    load_id SERIAL PRIMARY KEY,
    subject_id INT NOT NULL REFERENCES subjects(subject_id) ON DELETE CASCADE,
    semester_num INT NOT NULL,
    zet_value FLOAT8
);

CREATE TABLE IF NOT EXISTS subject_controls (
    control_id SERIAL PRIMARY KEY,
    subject_id INT NOT NULL REFERENCES subjects(subject_id) ON DELETE CASCADE,
    type_id INT NOT NULL REFERENCES control_types(type_id) ON DELETE CASCADE,
    semester_num INT NOT NULL
);

CREATE TABLE IF NOT EXISTS teachers (
    teacher_id INTEGER PRIMARY KEY,
    role user_role NOT NULL DEFAULT 'teacher' CHECK (role = 'teacher'),
    name VARCHAR(100) NOT NULL,
    lectern_id INT REFERENCES lecterns(lectern_id) ON DELETE SET NULL,
    job_title VARCHAR(50),
    FOREIGN KEY (teacher_id, role) REFERENCES users(id, role) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS groups (
    group_id SERIAL PRIMARY KEY,
    group_name VARCHAR(30) NOT NULL,
    lectern_id INT REFERENCES lecterns(lectern_id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS students (
    student_id INTEGER PRIMARY KEY,
    role user_role NOT NULL DEFAULT 'student' CHECK (role = 'student'),
    student_name VARCHAR(100),
    group_id INT REFERENCES groups(group_id) ON DELETE CASCADE,
    FOREIGN KEY (student_id, role) REFERENCES users(id, role) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS attendance_sessions (
    session_id SERIAL PRIMARY KEY,
    teacher_id INTEGER NOT NULL REFERENCES teachers(teacher_id) ON DELETE CASCADE,
    subject_id INTEGER NOT NULL REFERENCES subjects(subject_id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS attendance_marks (
    session_id INTEGER REFERENCES attendance_sessions(session_id) ON DELETE CASCADE,
    student_id INTEGER REFERENCES students(student_id) ON DELETE CASCADE,
    marked_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (session_id, student_id)
);

INSERT INTO control_types (type_name)
SELECT unnest(ARRAY['Экзамен', 'Зачет', 'Зачет с оц.'])
ON CONFLICT (type_name) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS attendance_marks;
DROP TABLE IF EXISTS attendance_sessions;
DROP TABLE IF EXISTS students;
DROP TABLE IF EXISTS groups;
DROP TABLE IF EXISTS teachers;
DROP TABLE IF EXISTS subject_controls;
DROP TABLE IF EXISTS semester_load;
DROP TABLE IF EXISTS subject_metrics;
DROP TABLE IF EXISTS subjects;
DROP TABLE IF EXISTS control_types;
DROP TABLE IF EXISTS lecterns;
DROP TABLE IF EXISTS users;
DROP TYPE IF EXISTS user_role;
-- +goose StatementEnd
