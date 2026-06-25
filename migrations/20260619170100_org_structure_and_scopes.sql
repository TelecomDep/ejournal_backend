-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Факультет — верхний уровень оргструктуры (охват декана).
-- Иерархия: faculties -> lecterns (кафедры) -> groups -> students;
-- teachers привязаны к lecterns.
CREATE TABLE IF NOT EXISTS faculties (
    faculty_id SERIAL PRIMARY KEY,
    code VARCHAR(20) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL
);

ALTER TABLE lecterns
    ADD COLUMN IF NOT EXISTS faculty_id INT REFERENCES faculties(faculty_id) ON DELETE SET NULL;

-- Привязка надзорной роли (head/dean) к её охвату: head -> кафедра,
-- dean -> факультет. Одна запись на единицу охвата.
CREATE TABLE IF NOT EXISTS org_scopes (
    scope_id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    lectern_id INT REFERENCES lecterns(lectern_id) ON DELETE CASCADE,
    faculty_id INT REFERENCES faculties(faculty_id) ON DELETE CASCADE,
    CONSTRAINT org_scope_target CHECK (lectern_id IS NOT NULL OR faculty_id IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS idx_org_scopes_user ON org_scopes (user_id);

-- Демо-факультет и привязка всех существующих кафедр к нему.
INSERT INTO faculties (code, name)
VALUES ('FIT', 'Факультет информационных технологий')
ON CONFLICT (code) DO NOTHING;

UPDATE lecterns
SET faculty_id = (SELECT faculty_id FROM faculties WHERE code = 'FIT')
WHERE faculty_id IS NULL;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
DECLARE
    v_faculty_id integer;
    v_lectern_id integer;
    v_head_id integer;
    v_dean_id integer;
BEGIN
    SELECT faculty_id INTO v_faculty_id FROM faculties WHERE code = 'FIT' LIMIT 1;

    -- Кафедра для зав. кафедрой: берём ту, где больше всего студентов,
    -- чтобы у демо-аккаунта был осмысленный охват.
    SELECT l.lectern_id
    INTO v_lectern_id
    FROM lecterns l
    LEFT JOIN groups g ON g.lectern_id = l.lectern_id
    LEFT JOIN students s ON s.group_id = g.group_id
    GROUP BY l.lectern_id
    ORDER BY COUNT(s.student_id) DESC, l.lectern_id ASC
    LIMIT 1;

    -- Демо-аккаунт заведующего кафедрой (head_test / 123456).
    INSERT INTO users (login, password_hash, role)
    VALUES ('head_test', crypt('123456', gen_salt('bf', 10)), 'head')
    ON CONFLICT (login) DO UPDATE
        SET password_hash = EXCLUDED.password_hash,
            role = EXCLUDED.role
    RETURNING id INTO v_head_id;

    IF v_head_id IS NULL THEN
        SELECT id INTO v_head_id FROM users WHERE login = 'head_test' LIMIT 1;
    END IF;

    DELETE FROM org_scopes WHERE user_id = v_head_id;
    IF v_lectern_id IS NOT NULL THEN
        INSERT INTO org_scopes (user_id, lectern_id) VALUES (v_head_id, v_lectern_id);
    END IF;

    -- Демо-аккаунт декана (dean_test / 123456).
    INSERT INTO users (login, password_hash, role)
    VALUES ('dean_test', crypt('123456', gen_salt('bf', 10)), 'dean')
    ON CONFLICT (login) DO UPDATE
        SET password_hash = EXCLUDED.password_hash,
            role = EXCLUDED.role
    RETURNING id INTO v_dean_id;

    IF v_dean_id IS NULL THEN
        SELECT id INTO v_dean_id FROM users WHERE login = 'dean_test' LIMIT 1;
    END IF;

    DELETE FROM org_scopes WHERE user_id = v_dean_id;
    IF v_faculty_id IS NOT NULL THEN
        INSERT INTO org_scopes (user_id, faculty_id) VALUES (v_dean_id, v_faculty_id);
    END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM users WHERE login IN ('head_test', 'dean_test');
DROP TABLE IF EXISTS org_scopes;
ALTER TABLE lecterns DROP COLUMN IF EXISTS faculty_id;
DROP TABLE IF EXISTS faculties;
-- +goose StatementEnd
