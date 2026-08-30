-- +goose Up
-- Repair existing databases where the presentation data was created after the
-- original head_test scope and therefore remained outside department analytics.
DELETE FROM org_scopes
WHERE user_id = (SELECT id FROM users WHERE login = 'head_test')
  AND lectern_id IS NOT NULL;

INSERT INTO org_scopes (user_id, lectern_id)
SELECT u.id, l.lectern_id
FROM users u
JOIN lecterns l ON l.code = 'DEMO-IT'
WHERE u.login = 'head_test'
  AND u.role = 'head';

UPDATE teachers t
SET lectern_id = l.lectern_id
FROM users u, lecterns l
WHERE t.user_id = u.id
  AND u.login = 'head_test'
  AND u.role = 'head'
  AND l.code = 'DEMO-IT';

-- +goose Down
DELETE FROM org_scopes scope
USING users u, lecterns l
WHERE scope.user_id = u.id
  AND scope.lectern_id = l.lectern_id
  AND u.login = 'head_test'
  AND l.code = 'DEMO-IT';

INSERT INTO org_scopes (user_id, lectern_id)
SELECT u.id, l.lectern_id
FROM users u
JOIN lecterns l ON l.code = 'KAF-IT'
WHERE u.login = 'head_test'
  AND NOT EXISTS (
      SELECT 1
      FROM org_scopes scope
      WHERE scope.user_id = u.id
        AND scope.lectern_id = l.lectern_id
  );
