# EJournal Backend (Go)

Небольшой backend-сервис для электронного журнала: регистрация, логин, профиль пользователя и отметка посещаемости через одноразовую ссылку.

## Что внутри

- REST API на `Fiber` (`:8888`)
- JWT-аутентификация
- Роли: `teacher` и `student`
- Генерация инвайт-ссылки на посещаемость (только для преподавателя)
- Подтверждение посещаемости по токену (только для студента)
- Внутренний worker pool для обработки запросов

## Стек

- Go `1.26.1`
- `github.com/gofiber/fiber/v2`
- `github.com/golang-jwt/jwt/v5`

## Быстрый старт

1. Создайте `.env` из шаблона:

```bash
cp .env.example .env
```

2. Установите переменные окружения (или отредактируйте `.env`):

```powershell
$env:JWT_SECRET="поменяй---------------------------------"
$env:SITE_BASE_URL="http://localhost:3000"
$env:APP_PORT="8888"
$env:CORS_ALLOW_ORIGINS="http://localhost:3000,http://127.0.0.1:3000"
```

3. Запустите сервис:

```powershell
go run ./cmd/server
```

Сервер стартует на `http://localhost:8888`.

## Переменные окружения

- `JWT_SECRET` (обязательно): ключ подписи JWT
- `SITE_BASE_URL` (необязательно): базовый URL фронтенда для формирования ссылки приглашения  
  По умолчанию: `http://localhost:3000`
- `APP_PORT` (необязательно): порт HTTP-сервера  
  По умолчанию: `8888`
- `CORS_ALLOW_ORIGINS` (необязательно): список origin через запятую для CORS  
  По умолчанию: `http://localhost:3000,http://127.0.0.1:3000`
- `DB_DSN` (обязательно): строка подключения PostgreSQL.  
  Пример: `postgres://postgres:postgres@localhost:5432/ejournal?sslmode=disable`

Сервис работает только с PostgreSQL: `register/login/profile` и посещаемость пишутся/читаются из БД.

## Goose миграции

Добавлена миграция схемы БД: `migrations/20260412152000_init_parser_schema.sql`.

Пример запуска:

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
goose -dir migrations postgres "postgres://postgres:postgres@localhost:5432/ejournal?sslmode=disable" up
```

## Docker

Сборка образа:

```bash
docker build -t ejournal-backend .
```

Запуск контейнера с env-файлом:

```bash
docker run -d --name ejournal-backend --env-file .env -p 8888:8888 ejournal-backend
```

## Docker Compose

Запуск:

```bash
docker compose up -d --build
```

Что происходит при старте:
- поднимается `postgres`
- ожидается `healthcheck` БД
- запускается `migrate` и применяет `goose up`
- только после успешной миграции стартует `ejournal-backend`
- автоматически создаются тестовые пользователи:
  - `teacher_test` / `123456`
  - `student_test` / `123456`
- автоматически создается тестовый предмет:
  - `subject_index=TEST-001`, `name=Networks`

Данные Postgres сохраняются в именованном volume `pgdata`.

Остановка:

```bash
docker compose down
```

Полное удаление БД-данных (осторожно):

```bash
docker compose down -v
```

## API

### 1) Регистрация

`POST /register`

```json
{
  "login": "teacher1",
  "password": "123456",
  "role": "teacher"
}
```

### 2) Логин

`POST /login`

```json
{
  "login": "teacher1",
  "password": "123456"
}
```

В ответе приходит `token`, используйте его в `Authorization: Bearer <token>`.

### 3) Профиль

`GET /profile`

### 4) Создать ссылку посещаемости (teacher)

`POST /api/teacher/attendance-link`

или алиас для мобильного/веб-приложения:

`POST /api/teacher/attendance/session`

```json
{
  "subject_id": 1,
  "group_ids": [1, 2],
  "lesson_name": "Networks",
  "expires_minutes": 20
}
```

`subject_id` и `group_ids` обязательны при работе с Postgres.

Пример успешного ответа:

```json
{
  "id": "http-attendance-link",
  "ok": true,
  "result": {
    "lesson_id": "1",
    "subject_id": 1,
    "lesson_name": "Networks",
    "invite_token": "<token>",
    "url": "http://localhost:3000/attendance/join?token=<token>",
    "join_url": "http://localhost:3000/attendance/join?token=<token>",
    "qr_payload": "http://localhost:3000/attendance/join?token=<token>",
    "group_ids": [1, 2],
    "roster_size": 35,
    "teacher_id": "1",
    "expires_at": "2026-04-15T15:00:00Z",
    "expires_minutes": 20
  }
}
```

Поля для приложения:
- `join_url` — ссылка, которую можно открыть в WebView/браузере
- `qr_payload` — строка для генерации QR-кода

### 5) Подтвердить посещаемость (student)

`POST /api/student/attendance/confirm`

```json
{
  "invite_token": "<token>"
}
```

### 6) Просмотр посещаемости по группе (teacher)

`POST /api/teacher/attendance/group`

```json
{
  "group_id": 1,
  "subject_id": 1
}
```

`subject_id` опционален. Если его не передать, статистика будет по всем предметам преподавателя для выбранной группы.

## Примеры curl

```bash
# Register teacher
curl -X POST http://localhost:8888/register \
  -H "Content-Type: application/json" \
  -d '{"login":"teacher1","password":"123456","role":"teacher"}'

# Login teacher
curl -X POST http://localhost:8888/login \
  -H "Content-Type: application/json" \
  -d '{"login":"teacher1","password":"123456"}'

# Profile
curl http://localhost:8888/profile \
  -H "Authorization: Bearer <TOKEN>"
  #вставьте токен который выдался выше после логина
```

## Структура проекта

- `cmd/server/main.go` - точка входа приложения
- `internal/app/service.go` - доменная логика, JWT, роли, worker pool
- `internal/httpserver/server.go` - HTTP-слой и маршруты
- `internal/config/config.go` - загрузка конфигурации из env
- `internal/db/*` - слой доступа к PostgreSQL (store + репозитории)
- `migrations/*` - goose-миграции БД
- `go.mod` / `go.sum` - зависимости
