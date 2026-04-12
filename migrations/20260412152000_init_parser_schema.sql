-- +goose Up
-- +goose StatementBegin
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
    teacher_id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    lectern_id INT REFERENCES lecterns(lectern_id) ON DELETE SET NULL,
    job_title VARCHAR(50)
);

CREATE TABLE IF NOT EXISTS groups (
    group_id SERIAL PRIMARY KEY,
    group_name VARCHAR(30) NOT NULL,
    lectern_id INT REFERENCES lecterns(lectern_id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS students (
    student_id SERIAL PRIMARY KEY,
    student_name VARCHAR(100),
    group_id INT REFERENCES groups(group_id) ON DELETE CASCADE
);

INSERT INTO control_types (type_name)
SELECT unnest(ARRAY['Экзамен', 'Зачет', 'Зачет с оц.'])
ON CONFLICT (type_name) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS students;
DROP TABLE IF EXISTS groups;
DROP TABLE IF EXISTS teachers;
DROP TABLE IF EXISTS subject_controls;
DROP TABLE IF EXISTS semester_load;
DROP TABLE IF EXISTS subject_metrics;
DROP TABLE IF EXISTS subjects;
DROP TABLE IF EXISTS control_types;
DROP TABLE IF EXISTS lecterns;
-- +goose StatementEnd
