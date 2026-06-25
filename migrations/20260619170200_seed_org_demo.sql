-- +goose Up
-- +goose StatementBegin
-- Демо-оргструктура: в showcase-данных не было ни одной кафедры, поэтому
-- охваты head/dean оказывались пустыми. Создаём демо-кафедру под факультетом
-- FIT, привязываем к ней «бесхозные» группы и преподавателей и назначаем её
-- заведующему (head_test).
DO $$
DECLARE
    v_faculty_id integer;
    v_lectern_id integer;
    v_head_id integer;
BEGIN
    SELECT faculty_id INTO v_faculty_id FROM faculties WHERE code = 'FIT' LIMIT 1;

    -- Демо-кафедра (idempotent по code).
    SELECT lectern_id INTO v_lectern_id FROM lecterns WHERE code = 'KAF-IT' LIMIT 1;
    IF v_lectern_id IS NULL THEN
        INSERT INTO lecterns (code, name, faculty_id)
        VALUES ('KAF-IT', 'Кафедра инфокоммуникационных технологий', v_faculty_id)
        RETURNING lectern_id INTO v_lectern_id;
    ELSE
        UPDATE lecterns SET faculty_id = v_faculty_id WHERE lectern_id = v_lectern_id;
    END IF;

    -- Привязываем «бесхозные» группы и преподавателей к демо-кафедре.
    UPDATE groups SET lectern_id = v_lectern_id WHERE lectern_id IS NULL;
    UPDATE teachers SET lectern_id = v_lectern_id WHERE lectern_id IS NULL;

    -- Назначаем head_test заведующим демо-кафедрой.
    SELECT id INTO v_head_id FROM users WHERE login = 'head_test' LIMIT 1;
    IF v_head_id IS NOT NULL THEN
        DELETE FROM org_scopes WHERE user_id = v_head_id;
        INSERT INTO org_scopes (user_id, lectern_id) VALUES (v_head_id, v_lectern_id);
    END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM org_scopes WHERE user_id = (SELECT id FROM users WHERE login = 'head_test');
UPDATE groups SET lectern_id = NULL
    WHERE lectern_id = (SELECT lectern_id FROM lecterns WHERE code = 'KAF-IT');
UPDATE teachers SET lectern_id = NULL
    WHERE lectern_id = (SELECT lectern_id FROM lecterns WHERE code = 'KAF-IT');
DELETE FROM lecterns WHERE code = 'KAF-IT';
-- +goose StatementEnd
