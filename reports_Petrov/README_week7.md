# Неделя 7

## Переход на DB-only режим, автоматизация запуска БД и стабилизация attendance flow

### 1) Переход с in-memory на PostgreSQL-only

На этой неделе приложение переведено в режим работы только с БД.

Что сделано:
- удалены in-memory ветки в `internal/app/service.go`
- регистрация, логин, профиль и подтверждение посещаемости выполняются через репозитории PostgreSQL
- в `cmd/server/main.go` сделан обязательный `DB_DSN` (без него сервер не стартует)

Практический эффект:
- единый источник данных
- одинаковое поведение между рестартами сервера
- отсутствие расхождения между in-memory и DB сценариями

### 2) Обновление схемы БД под роли и посещаемость

Расширена базовая миграция `migrations/20260412152000_init_parser_schema.sql`.

Добавлено:
- `user_role` (ENUM)
- `users` (логин, хеш пароля, роль)
- связь `teachers/students` с `users(id, role)`
- `attendance_sessions` и `attendance_marks`

Также добавлены новые репозитории:
- `internal/db/user_repository.go`
- `internal/db/attendance_repository.go`

И подключены в `Store`:
- `Users`
- `Attendance`

### 3) Автозапуск БД и миграций через Docker Compose

Обновлен `docker-compose.yml`:
- добавлен сервис `postgres` с `healthcheck`
- добавлен сервис `migrate`, выполняющий `goose up`
- `ejournal-backend` запускается только после успешной миграции
- добавлен постоянный volume `pgdata`

Уточнены важные детали:
- внутри docker-сети используется `postgres:5432`
- для хоста можно использовать `localhost:<POSTGRES_PORT>` (например `5433`)
- исправлена подстановка DSN для мигратора (`"$$DB_DSN"` в compose command)

### 4) Seed-миграции для тестирования без ручной подготовки

Добавлены миграции:
- `migrations/20260417090000_seed_test_users.sql`
  - `teacher_test / 123456`
  - `student_test / 123456`
- `migrations/20260417091000_seed_test_subject.sql`
  - тестовый предмет `TEST-001 / Networks`

Практический эффект:
- команда может сразу тестировать API без ручной регистрации и ручного bootstrap данных

### 5) Проверка и отладка ручек посещаемости

Проверен сценарий:
1. `POST /login` преподавателя
2. `POST /api/teacher/attendance-link`
3. `POST /login` студента
4. `POST /api/student/attendance/confirm`

Результат:
- генерация `join_url` и `invite_token` подтверждена
- подтверждение посещаемости успешно записывается в `attendance_marks`

Выявленная проблема:
- `subjects.subject_index` nullable в БД, но в коде сканируется в non-null `string`
- из-за этого возникала ошибка `failed to load subject` при `NULL`

## План на следующую неделю

- добавить отдельную `ALTER`-миграцию для безопасного апгрейда со старой схемы на новую
- исправить nullable-модель `subject_index` в Go-коде
- добавить HTTP endpoints для CRUD по `subjects`, чтобы не использовать SQL вручную
