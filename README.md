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

- Go `1.21+`
- `github.com/gofiber/fiber/v2`
- `github.com/golang-jwt/jwt/v5`

## Быстрый старт

1. Установите переменные окружения:

```powershell
$env:JWT_SECRET="super-secret-key"
$env:SITE_BASE_URL="http://localhost:3000"
```

2. Запустите сервис:

```powershell
go run .
```

Сервер стартует на `http://localhost:8888`.

## Переменные окружения

- `JWT_SECRET` (обязательно): ключ подписи JWT
- `SITE_BASE_URL` (необязательно): базовый URL фронтенда для формирования ссылки приглашения  
  По умолчанию: `http://localhost:3000`

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

```json
{
  "lesson_name": "Networks",
  "expires_minutes": 20
}
```

### 5) Подтвердить посещаемость (student)

`POST /api/student/attendance/confirm`

```json
{
  "invite_token": "<token>"
}
```

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
```

## Структура проекта

- `main.go` - доменная логика, JWT, роли, worker pool
- `http.go` - HTTP-слой и маршруты
- `go.mod` / `go.sum` - зависимости