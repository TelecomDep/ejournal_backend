# Неделя 5

## Подготовка проекта к серверному деплою и улучшение безопасности

### Вынесение конфигурации в переменные окружения

На этой неделе конфигурация была централизована и вынесена в отдельный пакет `internal/config`.  
Теперь приложение читает настройки из env и не держит важные данные в коде.

Добавлена структура:

```go
type AppConfig struct {
	JWTSecret        string
	SiteBaseURL      string
	AppPort          string
	CORSAllowOrigins string
}
```

Загрузка происходит через `config.Load()`, где:
- `JWT_SECRET` обязателен
- `SITE_BASE_URL` имеет дефолт `http://localhost:3000`
- `APP_PORT` имеет дефолт `8888`
- `CORS_ALLOW_ORIGINS` имеет дефолт для локальной разработки

Это сделало запуск на сервере проще и безопаснее: настройки меняются без правок кода.

### Добавление `.env.example`

Чтобы стандартизировать запуск, добавлен файл-шаблон:

```env
JWT_SECRET=change-me-in-production
SITE_BASE_URL=http://localhost:3000
APP_PORT=8888
CORS_ALLOW_ORIGINS=http://localhost:3000,http://127.0.0.1:3000
```

Рабочие секреты теперь хранятся в `.env`, а в репозиторий попадает только шаблон.

### Dockerfile для контейнерного запуска

Добавлен multi-stage `Dockerfile` на Go `1.26.1`:

```dockerfile
FROM golang:1.26.1-alpine AS builder
...
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /build/app ./cmd/server
...
CMD ["./app"]
```

Это позволяет собирать компактный production-образ и запускать сервис без установки Go на сервере.

### Добавление `docker-compose.yml`

Для более простого управления добавлен `docker-compose.yml`:

```yaml
services:
  ejournal-backend:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: ejournal-backend
    env_file:
      - .env
    ports:
      - "${APP_PORT:-8888}:${APP_PORT:-8888}"
    restart: unless-stopped
```

Теперь сервер можно поднять одной командой:

```bash
docker compose up -d --build
```

### Хеширование паролей через bcrypt

Вместо хранения пароля в открытом виде реализовано хеширование через `bcrypt`.

При регистрации:

```go
hashedPassword, err := bcrypt.GenerateFromPassword([]byte(data.Password), bcrypt.DefaultCost)
```

При логине:

```go
err := bcrypt.CompareHashAndPassword([]byte(user.Pass), []byte(data.Password))
```

Это существенно повышает безопасность пользовательских данных.

### Рефакторинг структуры проекта

Код был разложен по папкам, чтобы проект выглядел чище и легче масштабировался:

- `cmd/server/main.go` — точка входа
- `internal/app/service.go` — бизнес-логика (JWT, роли, посещаемость, bcrypt, worker pool)
- `internal/httpserver/server.go` — HTTP-слой (Fiber-роуты и обработчики)
- `internal/config/config.go` — загрузка env-конфига

Старая плоская структура (`main.go`, `http.go`, `config.go` в корне) удалена.

### Обновление документации

`README.md` обновлён под текущую архитектуру:
- запуск через `go run ./cmd/server`
- добавлены Docker и Docker Compose команды
- описана новая структура проекта


### Планы на будущее:

- вынести in-memory хранилища в постоянную БД (PostgreSQL)
- добавить миграции
- добавить базовые тесты на регистрацию/логин/посещаемость
