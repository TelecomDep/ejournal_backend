-- +goose Up
-- Backfill registration_invites for existing students and teachers who lack invite records

INSERT INTO registration_invites (invite_code, role, student_id, used_at, created_at)
SELECT 
    UPPER(encode(gen_random_bytes(8), 'hex')),
    'student'::user_role,
    st.student_id,
    CASE WHEN st.user_id IS NOT NULL THEN NOW() ELSE NULL END,
    NOW()
FROM students st
WHERE NOT EXISTS (
    SELECT 1 FROM registration_invites ri WHERE ri.student_id = st.student_id
);

INSERT INTO registration_invites (invite_code, role, teacher_id, used_at, created_at)
SELECT 
    UPPER(encode(gen_random_bytes(8), 'hex')),
    'teacher'::user_role,
    t.teacher_id,
    CASE WHEN t.user_id IS NOT NULL THEN NOW() ELSE NULL END,
    NOW()
FROM teachers t
WHERE NOT EXISTS (
    SELECT 1 FROM registration_invites ri WHERE ri.teacher_id = t.teacher_id
);

-- +goose Down
-- Nothing to undo
