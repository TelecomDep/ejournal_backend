# EJournal Backend (Go)

Backend-сервис электронного журнала: регистрация и логин, посещаемость по QR/геолокации, оценки (control points), ролевая система с иерархией преподаватель → зав. кафедрой → декан → админ, загрузка аватаров, восстановление пароля по email.

## Что внутри

- REST API на `Fiber` (`:8888`), Swagger UI (`/swagger/index.html`)
- JWT-аутентификация
- Роли: `student`, `teacher`, `head` (зав. кафедрой), `dean` (декан), `admin`
- Посещаемость: преподаватель открывает сессию (QR/инвайт-токен с ограниченным сроком жизни), студент подтверждает по токену; для Android-клиента — с проверкой геолокации (не дальше 200м от точки преподавателя)
- Оценки: преподаватель создаёт контрольные точки по предмету и выставляет баллы, студент видит свои оценки по предмету, сводно по всем предметам и в виде radar-диаграммы успеваемости
- Supervisory overview (`/api/staff/overview`): преподаватель видит свои группы, зав. кафедрой — всех преподавателей/студентов своей кафедры, декан — всего своего факультета, админ — всё
- Отчёт успеваемости для зав. кафедрой и выше в Excel (`/api/staff/reports/performance.xlsx`) или PDF (`/api/staff/reports/performance.pdf`): рейтинг студентов по предметам с посещаемостью, по потоку и по каждой группе
- Регистрация по инвайт-коду (роль определяется записью в БД) и legacy-регистрация по общему `role_hash`
- Восстановление пароля по email (SMTP) и загрузка аватара
- Внутренний worker pool для обработки запросов

## Стек

- Go `1.26.1`
- `github.com/gofiber/fiber/v2`
- `github.com/golang-jwt/jwt/v5`
- `github.com/swaggo/swag` (Swagger)
- PostgreSQL + `pressly/goose` для миграций
- Frontend: React (см. `frontend/`)

## Быстрый старт

1. Создайте `.env` из шаблона:

```bash
cp .env.example .env
```

2. Установите переменные окружения (или отредактируйте `.env`):

```powershell
$env:JWT_SECRET="поменяй---------------------------------"
$env:SITE_BASE_URL="http://localhost:9001"
$env:APP_PORT="9999"
$env:CORS_ALLOW_ORIGINS="http://localhost:9001,http://127.0.0.1:9001"
$env:ROLE_HASH_TEACHER="TEACHER-HASH-2026"
$env:ROLE_HASH_STUDENT="STUDENT-HASH-2026"
$env:DEFAULT_STUDENT_GROUP_ID="1"
$env:ALLOW_EARLY_ATTENDANCE="true"
```

3. Запустите сервис:

```powershell
go run ./cmd/server
```

Сервер стартует на `http://localhost:9999`.

## Переменные окружения

- `JWT_SECRET` (обязательно): ключ подписи JWT
- `SITE_BASE_URL` (необязательно): базовый URL, используется в письмах (ссылка восстановления пароля) и для ссылок на аватары  
  По умолчанию: `http://localhost:3000`
- `APP_PORT` (необязательно): порт HTTP-сервера  
  По умолчанию: `8888` (в локальном `.env` используется `9999`)
- `CORS_ALLOW_ORIGINS` (необязательно): список origin через запятую для CORS  
  По умолчанию: `http://localhost:3000,http://127.0.0.1:3000`
- `DB_DSN` (обязательно): строка подключения PostgreSQL.  
  Пример: `postgres://postgres:postgres@localhost:5432/ejournal?sslmode=disable`
- `ROLE_HASH_TEACHER` / `ROLE_HASH_STUDENT` (нужны для legacy hash-регистрации): общие секреты для самостоятельной регистрации по роли  
  По умолчанию: `TEACHER-HASH-2026` / `STUDENT-HASH-2026`
- `DEFAULT_STUDENT_GROUP_ID` (необязательно): группа, которая назначается студенту при legacy-регистрации через `/register`  
  По умолчанию: `1`
- `ALLOW_EARLY_ATTENDANCE` (необязательно): если `true`, разрешает студенту подтвердить посещаемость до официального начала сессии  
  По умолчанию: `false` (в локальном `.env` включено `true` для тестов)
- `UPLOAD_DIR` (необязательно): каталог для загруженных файлов (аватары), отдаётся на `/uploads`  
  По умолчанию: `uploads`
- `SMTP_HOST` / `SMTP_PORT` / `SMTP_USER` / `SMTP_PASSWORD` / `SMTP_FROM` (необязательно): параметры почтового сервера для писем восстановления пароля.  
  Если `SMTP_HOST` не задан, письмо не отправляется — токен просто пишется в лог (удобно для локальной разработки)
- `DOMAIN` / `FRONTEND_PORT` / `FRONTEND_HTTPS_PORT` — см. раздел [HTTPS](#https-production)

Сервис работает только с PostgreSQL: все данные (пользователи, посещаемость, оценки, оргструктура) пишутся/читаются из БД.

## Роли и оргструктура

- `student` — видит только себя (свои оценки, посещаемость)
- `teacher` — свои группы и предметы
- `head` (зав. кафедрой) — все преподаватели и студенты своей кафедры (`lecterns` + `org_scopes`)
- `dean` (декан) — весь свой факультет (`faculties`)
- `admin` — всё без ограничений

## Goose миграции

Миграции лежат в `backend/migrations`. Ключевые этапы схемы: базовые сущности (пользователи, предметы, группы, посещаемость) → мультигрупповые сессии → расписание/журнал → инвайт-регистрация → оценки (`add_grades`) → email и токен восстановления пароля → роли `head`/`dean`/`admin` и оргструктура (`org_structure_and_scopes`).

Пример запуска вручную:

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
- поднимается `web` (nginx + фронтенд, проксирует API/uploads/swagger на backend)
- поднимается `mailserver` (SMTP для писем восстановления пароля)
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

## HTTPS (production)

Контейнер `web` (nginx) умеет сразу отдавать сайт по HTTPS с сертификатом Let's Encrypt.

Переменные окружения (`.env`):
- `DOMAIN` — домен сайта, по умолчанию `lms.signal.qlabs.pro`
- `FRONTEND_PORT` — HTTP-порт (редиректит на HTTPS), по умолчанию `9999`
- `FRONTEND_HTTPS_PORT` — HTTPS-порт, по умолчанию `443`

Первый запуск на новом домене/сервере:

```bash
./init-letsencrypt.sh
```

Скрипт сам поднимает временный сертификат, чтобы стартовал nginx, получает настоящий сертификат от Let's Encrypt через webroot-challenge и запускает `certbot` для автопродления (проверка каждые 12 часов, `web` перечитывает конфиг каждые 6 часов).

При смене домена: обновить `DOMAIN` в `.env`, выполнить `docker compose up -d --build web`, затем заново запустить `./init-letsencrypt.sh`.

## API

Swagger UI: `/swagger/index.html` (там же, где фронтенд — nginx проксирует `/swagger/` на бэкенд). Полный список ниже сгруппирован по областям; все `/api/**`-маршруты, кроме явно отмеченных, требуют заголовок `Authorization: Bearer <token>`.

### Аутентификация и аккаунт

| Метод и путь | Доступ | Описание |
|---|---|---|
| `POST /register` | без токена | legacy-регистрация по `role_hash` |
| `POST /register/by-invite` | без токена | регистрация по инвайт-коду, роль определяется записью в БД |
| `POST /login` | без токена | возвращает JWT |
| `GET /profile` | любой авторизованный | профиль текущего пользователя |
| `POST /api/auth/forgot-password` | без токена | отправляет письмо со ссылкой восстановления пароля |
| `POST /api/auth/reset-password` | без токена | подтверждает сброс пароля по токену из письма |
| `POST /api/user/email` | любой авторизованный | привязать/обновить email |
| `POST /api/user/upload-avatar` | любой авторизованный | загрузка аватара, `multipart/form-data`, поле `avatar` (jpg/png/webp, до 5 МБ) |

### Семестры

| Метод и путь | Доступ | Описание |
|---|---|---|
| `GET /api/semesters` | без токена | список всех семестров, включая закрытые и архивные |
| `GET /api/semesters/current` | без токена | открытый семестр, используемый по умолчанию |
| `POST /api/admin/semesters` | admin | создать семестр со статусом `planned` или `open` |
| `PATCH /api/admin/semesters/:semester_id/activate` | admin | открыть запланированный семестр и автоматически закрыть предыдущий |
| `PATCH /api/admin/semesters/:semester_id/close` | admin | закрыть открытый семестр и запретить изменение его данных |
| `PATCH /api/admin/semesters/:semester_id/archive` | admin | архивировать закрытый семестр, сохранив историческое чтение |

### Администратор: пользователи

| Endpoint | Доступ | Назначение |
|---|---|---|
| `GET /api/admin/users` | admin | список пользователей с пагинацией, поиском и фильтрами |
| `GET /api/admin/users/:user_id` | admin | получить одного пользователя |
| `POST /api/admin/users` | admin | создать пользователя и профиль его роли |
| `PATCH /api/admin/users/:user_id` | admin | изменить данные, роль или статус |
| `DELETE /api/admin/users/:user_id` | admin | мягко удалить пользователя (`status=archived`) |
| `GET /api/admin/stats` | admin, minister | системная статистика: пользователи по ролям, группы, семестр, Go-runtime |
| `GET /api/admin/org-structure` | admin, minister, director, dean | иерархическое дерево оргструктуры: факультеты → кафедры → группы |
| `GET /api/admin/roles` | admin | матрица ролевых прав и видимости данных (RBAC spectrum) |
| `PATCH /api/admin/roles/:role` | admin | динамическое переключение возможностей и прав роли |
| `GET /api/admin/antifraud/logs` | admin, minister, dean, director | журнал попыток обмана посещаемости и дублирования Device ID |
| `GET /api/admin/antifraud/top-cheaters` | admin, minister, dean, director | рейтинг студентов по количеству попыток читерства |
| `GET /api/admin/services` | admin, minister | Launchpad быстрой навигации: Grafana, Prometheus, Azimutt, Webmail |
| `GET /api/admin/audit-logs` | admin, minister | журнал административных действий и изменений конфигураций |
| `GET /api/admin/system/maintenance` | admin, minister | текущий статус режима технического обслуживания |
| `POST /api/admin/system/maintenance` | admin | переключатель режима Maintenance Mode (kill-switch) |
| `POST /api/user/device-token` | авториз. юзер | регистрация FCM Push-токена Android-устройства |
| `GET /api/user/device-tokens` | авториз. юзер | список зарегистрированных устройств пользователя |
| `DELETE /api/user/device-token` | авториз. юзер | отвязка Push-токена устройства |

### Двухфакторная аутентификация (2FA) и Push-уведомления
Если у пользователя включена двухфакторная аутентификация (`is_2fa_enabled = true`), при первичном входе через `/login` (без кода `two_fa_code`) бэкенд возвращает ответ `requires_2fa` и автоматически отправляет событийное событие Push-уведомления (Event-driven FCM Push) с текущим кодом TOTP на зарегистрированные Android-устройства пользователя. Данный подкод не требует фонового мониторинга 24/7 и оптимизирует энергопотребление смартфона.

Lifecycle семестра: `planned → open → closed → archived`. Одновременно может быть открыт только один семестр. Создание занятий, контрольных точек и изменение оценок разрешено только в открытом семестре и только между `starts_at` и `ends_at`. Закрытие и переключение блокируются, пока в текущем семестре есть активная attendance-сессия. При отсутствии `semester_id` backend использует открытый семестр; для исторического чтения можно передать ID закрытого или архивного периода.

`term_num` в таблице `semesters` означает календарную половину учебного года (`1` — осень, `2` — весна). Старый `semester_num` в учебной нагрузке и видах контроля остаётся номером семестра образовательной программы (`1…8`) и не подменяется календарным периодом.

### Посещаемость — преподаватель

| Метод и путь | Описание |
|---|---|
| `POST /api/teacher/attendance-link` (алиас `POST /api/teacher/attendance/session`) | создать сессию посещаемости + инвайт-ссылку/QR |
| `GET /api/teacher/attendance/session/marked-count?lesson_id=` | сколько студентов уже отметились |
| `GET /api/teacher/attendance/session/timer?lesson_id=` | оставшееся время сессии |
| `GET /api/teacher/attendance/session/active` | текущая активная сессия преподавателя |
| `GET /api/teacher/subjects` | предметы и группы преподавателя (из расписания) |
| `POST /api/teacher/attendance/group` | статистика посещаемости по группе (опционально по предмету) |
| `POST /api/teacher/attendance/student/history` | история посещаемости одного студента по предмету |
| `POST /api/teacher/attendance/mark` | ручная корректировка статуса студента в сессии (`present`/`absent`/`late`/`excused`) |
| `POST /api/teacher/group/performance` | сводный обзор посещаемости + оценок по группе |

### Посещаемость — студент

| Метод и путь | Описание |
|---|---|
| `POST /api/student/attendance/confirm` | подтвердить посещаемость по инвайт-токену |
| `GET /api/student/attendance/history?year=` | своя история посещаемости |
| `GET /api/student/schedule/day?date=` | своё расписание на день (текущая или следующая календарная неделя, по умолчанию — сегодня) |

### Android-совместимый API

| Метод и путь | Описание |
|---|---|
| `POST /lessons/create` | преподаватель создаёт занятие по названию предмета/групп + геолокация |
| `POST /api/student/mark-attendance` | студент отмечается по `lesson_id` + геолокация (проверка ≤200м от преподавателя) |

### Оценки

| Метод и путь | Доступ | Описание |
|---|---|---|
| `POST /api/teacher/grades/items` | teacher | создать контрольную точку по предмету |
| `POST /api/teacher/grades/items/list` | teacher | список контрольных точек по предмету |
| `DELETE /api/teacher/grades/items/:item_id` | owner/admin | мягко удалить контрольную точку; при оценках нужен `cascade=true` |
| `POST /api/teacher/grades/items/:item_id/restore` | owner/admin | восстановить контрольную точку |
| `POST /api/teacher/grades` | teacher | выставить/обновить балл студенту |
| `DELETE /api/teacher/grades/:grade_id` | owner/admin | мягко удалить ошибочный балл |
| `POST /api/teacher/grades/:grade_id/restore` | owner/admin | восстановить удаленный балл |
| `POST /api/teacher/grades/student` | teacher | оценочный лист одного студента |
| `POST /api/teacher/student/performance/radar` | teacher | radar-диаграмма успеваемости конкретного студента |
| `POST /api/student/grades` | student | свои оценки по предмету |
| `GET /api/student/performance/radar?semester_id=` | student | своя radar-диаграмма успеваемости за выбранный или открытый семестр |
| `GET /api/student/grades/all?semester_id=` | student | все предметы, оценки и сводная статистика за выбранный или открытый семестр |

### Supervisory overview

| Метод и путь | Описание |
|---|---|
| `GET /api/staff/overview` | обзор оргструктуры, объём зависит от роли (teacher/head/dean/admin) |
| `GET /api/staff/overview/students` | постраничный список студентов в зоне видимости: `page`, `page_size`, `group_id`, `search`, `sort`, `order` |
| `GET /api/staff/ratings/general?semester_id=` | JSON-выгрузка для общего рейтинга: кафедры, предметы, группы, студенты, согласия, посещаемость и оценки; без `semester_id` используется открытый семестр |
| `GET /api/staff/reports/performance.xlsx?semester_id=` | скачать Excel-отчёт успеваемости (только head/dean/admin): лист «Поток» по всему охвату + лист на каждую группу, студенты отсортированы по итоговому рейтингу, колонки — % по предметам, рейтинг, посещаемость, внизу средние значения |
| `GET /api/staff/reports/performance.pdf?semester_id=` | скачать тот же отчёт в PDF; без `semester_id` используется открытый семестр |

### Прочее

| Метод и путь | Описание |
|---|---|
| `GET /uploads/*` | статика (аватары и другие загруженные файлы) |
| `GET /swagger/*` | Swagger UI |
| `GET /healthz` | liveness-проверка backend |
| `GET /internal/metrics` | защищенные runtime/HTTP/SQL-метрики; требует `METRICS_TOKEN` и заголовок `X-Metrics-Token` |

### Примеры запросов

Регистрация преподавателя (legacy hash):

```json
POST /register
{
  "login": "teacher1",
  "password": "123456",
  "role_hash": "TEACHER-HASH-2026"
}
```

Регистрация по инвайт-коду (роль берётся из `registration_invites`):

```json
POST /register/by-invite
{
  "login": "student_login",
  "password": "StrongPassword123",
  "invite_code": "8D2C72771DF0"
}
```

Логин:

```json
POST /login
{
  "login": "teacher1",
  "password": "123456"
}
```

В ответе приходит `token`, используйте его в `Authorization: Bearer <token>`.

Создать сессию посещаемости:

```json
POST /api/teacher/attendance-link
{
  "subject_id": 1,
  "group_ids": [1, 2],
  "lesson_name": "Networks",
  "expires_minutes": 20
}
```

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
    "url": "http://localhost:3000/#/attendance/join?token=<token>",
    "join_url": "http://localhost:3000/#/attendance/join?token=<token>",
    "qr_payload": "http://localhost:3000/#/attendance/join?token=<token>",
    "group_ids": [1, 2],
    "roster_size": 35,
    "teacher_id": "1",
    "expires_at": "2026-04-15T15:00:00Z",
    "expires_minutes": 20
  }
}
```

`join_url` — ссылка для WebView/браузера, `qr_payload` — строка для генерации QR-кода.

Подтвердить посещаемость (студент):

```json
POST /api/student/attendance/confirm
{
  "invite_token": "<token>"
}
```

Ручная корректировка посещаемости (преподаватель, для своей сессии):

```json
POST /api/teacher/attendance/mark
{
  "lesson_id": 1,
  "student_id": 6,
  "status": "excused"
}
```

`status` — один из `present`, `absent`, `late`, `excused`.

Расписание студента на день (текущая или следующая неделя):

```
GET /api/student/schedule/day?date=2026-07-12
```

`date` необязателен (по умолчанию — сегодня), формат `YYYY-MM-DD`. Дата вне текущей/следующей календарной недели вернёт ошибку `400`.

Отметка студента из Android-приложения (с геолокацией):

```json
POST /api/student/mark-attendance
{
  "lesson_id": 1,
  "device_id": "android-device-id",
  "lat": 55.75,
  "lon": 37.61
}
```

## Примеры curl

```bash
# Register teacher
curl -X POST http://localhost:9999/register \
  -H "Content-Type: application/json" \
  -d '{"login":"teacher1","password":"123456","role_hash":"TEACHER-HASH-2026"}'

# Register student by student hash
curl -X POST http://localhost:9999/register \
  -H "Content-Type: application/json" \
  -d '{"login":"student_new","password":"123456","role_hash":"STUDENT-HASH-2026"}'

# Login teacher
curl -X POST http://localhost:9999/login \
  -H "Content-Type: application/json" \
  -d '{"login":"teacher1","password":"123456"}'

# Profile
curl http://localhost:9999/profile \
  -H "Authorization: Bearer <TOKEN>"
  #вставьте токен который выдался выше после логина

# Staff overview (роль head/dean/admin увидит больше, чем teacher)
curl http://localhost:9999/api/staff/overview \
  -H "Authorization: Bearer <TOKEN>"
```

## Frontend

React-приложение в `frontend/` (hash-роутинг через `useHashRoute`). Основные экраны и компоненты:

- `LoginPage`, `ProfilePage` / `PersonalAccount` / `ProfileSquare` — вход и профиль
- `TeacherAccount`, `AttendancePage`, `AttendanceGrid`, `AttendanceHeatmap`, `QRCode` — рабочее место преподавателя: сессии посещаемости, QR-код приглашения, тепловая карта посещаемости
- `GradesPage`, `StudentGradesPanel`, `RadarChart` — оценки и radar-диаграмма успеваемости
- `StaffDashboard` — обзор оргструктуры (роль-зависимый: teacher/head/dean/admin)
- `Calendar`, `DataTable`, `InfoCard`, `ThemeToggle` — общие UI-компоненты

## Структура проекта

Backend (`backend/`):
- `cmd/server/main.go` — точка входа приложения, Swagger-аннотации
- `internal/app/service.go` — доменная логика, JWT, роли, worker pool
- `internal/app/android.go` — посещаемость с геолокацией для Android-клиента
- `internal/app/grades.go` — контрольные точки и оценки
- `internal/app/semester.go` — lifecycle семестров и правила исторического чтения/записи
- `internal/app/supervision.go` — ролевой обзор оргструктуры (staff overview)
- `internal/app/schedule.go` — расписание студента на день
- `internal/app/report.go` — Excel-отчёт успеваемости (excelize)
- `internal/app/mailer.go` — отправка писем (SMTP, восстановление пароля)
- `internal/httpserver/server.go` — HTTP-слой и маршруты
- `internal/config/config.go` — загрузка конфигурации из env
- `internal/db/*` — слой доступа к PostgreSQL (store + репозитории)
- `docs/*` — сгенерированные swagger-файлы (`swag init`, не редактировать руками)
- `migrations/*` — goose-миграции БД
- `go.mod` / `go.sum` — зависимости

Frontend (`frontend/src/`):
- `pages/` — `AttendancePage`, `GradesPage`, `ProfilePage`
- `components/` — экраны и виджеты (см. раздел [Frontend](#frontend))
- `services/api.js` — обёртка над fetch
- `hooks/useHashRoute.js` — hash-роутинг

Инфраструктура (корень репозитория):
- `docker-compose.yml` — postgres, migrate, ejournal-backend, web (nginx+frontend), certbot, mailserver, prometheus, grafana, azimutt
- `frontend/nginx.conf.template` + `frontend/40-envsubst-domain.sh` — nginx-конфиг с доменом из `$DOMAIN`
- `init-letsencrypt.sh` — bootstrap первого Let's Encrypt сертификата
- `grafana/provisioning/` — автоматический провижининг Prometheus datasource и LMS Production Overview дашборда
- `scripts/manage_azimutt_users.py` — утилита управления разработчиками в Azimutt (активация плана Enterprise via DB trigger)

## Мониторинг Grafana и Azimutt

### Авто-провижининг Grafana
Система поставляется с готовыми конфигурациями провижининга Grafana (`grafana/provisioning/datasources` и `grafana/provisioning/dashboards`). При запуске Docker Compose дашборд **"LMS Production Overview"** становится сразу доступен без необходимости ручной настройки.
- URL: `https://<DOMAIN>:9001/grafana/`
- Метрики: Backend HTTP RPS, p95/p99 Latency, Go Goroutines/Allocated Memory, PostgreSQL Active Connections & Transactions.

### Управление Azimutt
Для предоставления разработчикам доступа к ERD-схемам Azimutt с разблокированным Enterprise-функционалом используется скрипт:
```bash
python3 scripts/manage_azimutt_users.py list
python3 scripts/manage_azimutt_users.py elevate user@example.com
```
Скрипт автоматически обновляет статус подписки в PostgreSQL Azimutt через внутренние триггеры.
