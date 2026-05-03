# Неделя 9

## Слияние frontend с main и запуск проекта через Docker Compose

### 1. Цель недели

На этой неделе основная задача была не добавлять новую бизнес-логику, а подготовить проект к дальнейшему CI/CD и собрать его в одно рабочее состояние:
- перенести актуальный frontend в ветку `main`
- сохранить backend-код из `main`
- настроить единый запуск backend, frontend, PostgreSQL и миграций через `docker compose`
- сделать так, чтобы frontend мог обращаться к backend без ручного запуска отдельных сервисов

То есть работа была подготовительным этапом перед автоматизацией сборки, проверки и деплоя.

### 2. Merge frontend в main

Ветка `ejournal-frontend` была слита в `main`.

При слиянии были конфликты, потому что frontend-ветка переносила React-проект в корень и удаляла часть backend-файлов. Конфликты были разрешены так:
- backend из `main` сохранен
- актуальный frontend оставлен в корне проекта
- старая папка `frontend/` удалена
- сгенерированный `build/` больше не хранится как основной способ запуска

Итог:
```bash
git merge ejournal-frontend
```

После разрешения конфликтов был создан merge-коммит:
```bash
Merge ejournal-frontend into main
```

### 3. Docker Compose для всего проекта

Был обновлен `docker-compose.yml`.

Теперь одной командой поднимаются:
- `postgres` - база данных
- `migrate` - применение goose-миграций
- `ejournal-backend` - Go backend
- `web` - React frontend через Nginx

Основной фрагмент:
```yaml
services:
  ejournal-backend:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "${APP_PORT:-8888}:${APP_PORT:-8888}"

  web:
    build:
      context: .
      dockerfile: Dockerfile.frontend
    depends_on:
      - ejournal-backend
    ports:
      - "${FRONTEND_PORT:-9999}:80"
```

Запуск проекта:
```bash
docker compose up -d --build
```

### 4. Dockerfile для frontend

Для frontend был добавлен отдельный `Dockerfile.frontend`.

Он собирает React-приложение внутри контейнера на Node 20, а затем отдает готовую сборку через Nginx.

```dockerfile
FROM node:20-alpine AS builder

WORKDIR /app

COPY package*.json ./
RUN npm ci

COPY public ./public
COPY src ./src
RUN npm run build

FROM nginx:1.27-alpine

COPY nginx.conf /etc/nginx/nginx.conf
COPY --from=builder /app/build /usr/share/nginx/html
```

Практический эффект:
- не нужно вручную запускать `npm run build` перед Docker
- frontend собирается одинаково на любой машине
- проблема со старой локальной версией Node не мешает Docker-запуску

### 5. Настройка Nginx-прокси

Был обновлен `nginx.conf`.

Frontend теперь открывается на `http://localhost:9999`, а API-запросы проксируются во внутренний Docker-сервис backend.

```nginx
location /api/ {
    proxy_pass http://ejournal-backend:8888;
}

location ~ ^/(login|profile|register)(/.*)?$ {
    proxy_pass http://ejournal-backend:8888;
}
```

Это убрало необходимость обращаться из браузера напрямую к разным адресам backend/frontend.

### 6. Исправление API URL во frontend

В `src/services/api.js` изменена логика выбора backend URL.

Теперь, если `REACT_APP_BACKEND_URL` не задан, frontend использует текущий origin страницы:

```javascript
const DEFAULT_BACKEND_URL = typeof window !== 'undefined'
  ? window.location.origin
  : 'http://localhost:8888';

const BACKEND_URL = (
  process.env.REACT_APP_BACKEND_URL || DEFAULT_BACKEND_URL
).replace(/\/$/, '');
```

Практический эффект:
- при открытии `http://localhost:9999` запросы идут через Nginx
- не нужно вручную прописывать backend URL для Docker-сценария
- локальная разработка через `.env` все еще возможна

### 7. Проверка результата

Были проверены:
```bash
docker compose config
docker compose build
docker compose up -d
```

Контейнеры после запуска:
```text
ejournal-postgres   Up / healthy
ejournal-backend    Up / :8888
sibsutis_front      Up / :9999
```

Основные адреса:
- frontend: `http://localhost:9999`
- backend: `http://localhost:8888`

### 8. Итог недели

Проект теперь можно запускать как единую систему через Docker Compose.

Главный результат:
- frontend слит в `main`
- backend сохранен
- PostgreSQL и миграции поднимаются автоматически
- frontend собирается в Docker
- Nginx раздает frontend и проксирует API в backend

Это создает основу для CI/CD:
- pipeline сможет собирать backend Docker-образ
- pipeline сможет собирать frontend Docker-образ
- можно будет запускать проверку `docker compose config`
- можно будет автоматически поднимать тестовый стенд через `docker compose up`

Пример будущих шагов в CI:
```yaml
- name: Validate compose
  run: docker compose config

- name: Build images
  run: docker compose build
```
