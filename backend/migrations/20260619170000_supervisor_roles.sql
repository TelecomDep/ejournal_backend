-- +goose NO TRANSACTION
-- +goose Up
-- Зав. кафедрой (head) и декан (dean) — новые надзорные роли поверх
-- существующих student/teacher/admin. ALTER TYPE ... ADD VALUE нельзя
-- выполнять внутри транзакции, поэтому файл помечен NO TRANSACTION.
ALTER TYPE user_role ADD VALUE IF NOT EXISTS 'head';
ALTER TYPE user_role ADD VALUE IF NOT EXISTS 'dean';

-- +goose Down
-- PostgreSQL не умеет удалять значения enum, поэтому откат — no-op.
SELECT 1;
