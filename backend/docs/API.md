# EJournal Backend API — документация для фронтенда

Backend API электронного журнала: авторизация, посещаемость по QR/ссылке, оценки, расписание, отчёты.

- **Base URL (локально):** `http://localhost:8888`
- **Формат:** JSON (кроме загрузки аватара и скачивания отчётов)
- **Swagger UI:** `GET /swagger/index.html`

## Общие сведения

### Авторизация

Все ручки, кроме `/register`, `/register/by-invite`, `/login`, `/api/auth/forgot-password`, `/api/auth/reset-password`, требуют JWT-токен в заголовке:

```
Authorization: Bearer <jwt_token>
```

Токен выдаётся ручками `/login` и `/register`.

### Единый формат ответа

Почти все ручки возвращают конверт:

```json
{
  "ok": true,
  "id": "http-login",
  "result": { ... },
  "error": ""
}
```

| Поле | Тип | Описание |
|---|---|---|
| `ok` | bool | `true` — успех, `false` — ошибка |
| `id` | string | Идентификатор операции (для отладки) |
| `result` | object/array | Полезные данные (при успехе) |
| `error` | string | Текст ошибки (при `ok: false`) |

### Коды ошибок

| Код | Значение |
|---|---|
| `400` | Невалидное тело запроса / параметры |
| `401` | Нет токена или токен невалиден |
| `403` | Роль не имеет доступа к ручке |
| `404` | Объект не найден |
| `409` | Конфликт (например, посещение уже отмечено) |
| `500` | Внутренняя ошибка сервера |

### Роли

Один пользователь может иметь несколько ролей. Поддерживаются `student`, `teacher`, `secretary`, `head`, `program_creator`, `director`, `dean`, `minister`, `admin`. Поле `role` содержит активную роль текущей сессии, `primary_role` — основную роль пользователя, `roles` — все назначенные роли.

---

## 1. Аутентификация

### POST `/register` — регистрация

Регистрация по одноразовому `invite_code`. Легаси-вариант с `role_hash` тоже поддерживается.

**Auth:** не требуется.

**Тело запроса:**

```json
{
  "login": "teacher1",
  "password": "secret",
  "invite_code": "ABC123",
  "role": "teacher",       // только для легаси-варианта
  "role_hash": "..."       // только для легаси-варианта
}
```

**Ответ 200:**

```json
{
  "ok": true,
  "id": "http-register",
  "result": {
    "login": "teacher1",
    "role": "teacher",
    "token": "<jwt_token>",
    "user_id": "5"
  }
}
```

**Ошибки:** `400`, `500`.

---

### POST `/register/by-invite` — регистрация по инвайт-коду

Создаёт аккаунт студента / преподавателя / админа по одноразовому коду из БД. Если клиент передаёт `agreement`, он должен принять текущую версию документа (`2026-09-01`); запись согласия сохраняется атомарно с аккаунтом и использованием инвайта. Старые клиенты могут не передавать это поле. Токен возвращается в ответе.

**Auth:** не требуется.

**Тело запроса:**

```json
{
  "invite_code": "ABC123",
  "login": "student_iks_21",
  "password": "secret",
  "agreement": {
    "version": "2026-09-01",
    "decision": "accepted"
  }
}
```

**Ответ 200 (`result`):**

```json
{
  "user_id": "12",
  "login": "student_iks_21",
  "role": "student",
  "group_id": 237,
  "group_name": "ИКС-433",
  "student_id": 56,
  "student_name": "Демин Сергей А.",
  "teacher_id": 3,
  "teacher_name": "Солодов Павел Сергеевич",
  "job_title": "Преподаватель"
}
```

Поля `student_*` / `teacher_*` / `group_*` заполняются в зависимости от роли.

**Ошибки:** `400`, `409` (логин занят / код использован), `500`.

---

### POST `/login` — вход

**Auth:** не требуется.

**Тело запроса:**

```json
{
  "login": "teacher_test",
  "password": "123456"
}
```

**Ответ 200 (`result`):**

```json
{
  "login": "teacher_test",
  "role": "teacher",
  "active_role": "teacher",
  "primary_role": "teacher",
  "roles": ["teacher", "head"],
  "token": "<jwt_token>",
  "user_ID": "3"
}
```

**Ошибки:** `401` (неверный логин/пароль), `500`.

---

### POST `/api/auth/switch-role` — переключить активную роль

**Auth:** Bearer.

```json
{ "role": "teacher" }
```

Сервер проверяет, что роль назначена пользователю, и возвращает новый JWT с подписанной активной ролью. Неназначенная роль возвращает `403`.

---

### GET `/profile` — профиль текущего пользователя

**Auth:** Bearer.

**Ответ 200 (`result`):**

```json
{
  "user_id": "3",
  "login": "student_iks_21",
  "role": "student",
  "active_role": "student",
  "primary_role": "student",
  "roles": ["student"],
  "name": "Демин Сергей А.",
  "avatar": "https://server.com/uploads/avatars/avatar.png",
  "group": "ИКС-433",
  "group_id": 237,
  "group_name": "ИКС-433",
  "student_id": 56,
  "student_name": "Демин Сергей А.",
  "teacher_id": 3,
  "teacher_name": "Солодов Павел Сергеевич",
  "job_title": "Преподаватель",
  "lectern_id": 1,
  "nfc_tag": "04:XX:YY:ZZ",
  "total_cheat_attempts": 0
}
```

Набор заполненных полей зависит от роли (у студента — `student_*`/`group_*`, у преподавателя — `teacher_*`/`job_title`/`lectern_id`).

**Ошибки:** `401`, `500`.

---

### POST `/api/auth/forgot-password` — запрос сброса пароля

Генерирует токен сброса и отправляет на email пользователя.

**Auth:** не требуется.

**Тело запроса:**

```json
{ "identity": "login_или_email" }
```

**Ответ 200:** стандартный конверт. **Ошибки:** `400`, `500`.

---

### POST `/api/auth/reset-password` — сброс пароля

**Auth:** не требуется.

**Тело запроса:**

```json
{
  "token": "<reset_token_из_письма>",
  "new_password": "newsecret"
}
```

**Ответ 200:** стандартный конверт. **Ошибки:** `400` (токен невалиден/просрочен), `500`.

---

## 2. Профиль пользователя

### POST `/api/user/email` — привязать/обновить email

**Auth:** Bearer.

**Тело запроса:**

```json
{ "email": "user@example.com" }
```

**Ответ 200:** стандартный конверт. **Ошибки:** `400`, `401`, `500`.

---

### POST `/api/user/upload-avatar` — загрузка аватара

**Auth:** Bearer. **Content-Type:** `multipart/form-data`.

**Форма:** поле `avatar` — файл JPEG / PNG / WebP, до 5 MiB.

**Ответ 200 (`result`):** публичный URL картинки (отдаётся со статики `/uploads/...`).

**Ошибки:** `400` (не тот формат / слишком большой), `401`.

---

## 3. Посещаемость — ведущий занятия

Маршруты `/api/teaching/*` доступны ролям `teacher` и `head` при наличии личного преподавательского профиля и назначения на дисциплину. Старый префикс `/api/teacher/*` временно сохранён как совместимый алиас.

### POST `/api/teaching/attendance-link` — создать сессию посещаемости

Алиас: `POST /api/teaching/attendance/session` (то же самое).

Преподаватель создаёт сессию и получает ссылку/QR для отметки. Если `subject_id` / `group_ids` не переданы — берутся из ближайшей пары в расписании.

**Auth:** Bearer (`teacher` или `head`).

**Тело запроса:**

```json
{
  "lesson_name": "Networks",
  "expires_minutes": 20,
  "subject_id": 2,        // опционально
  "group_ids": [1],       // опционально
  "lesson_id": 5          // опционально
}
```

**Ответ 200 (`result`):**

```json
{
  "lesson_id": "5",
  "lesson_name": "Networks",
  "subject_id": 2,
  "teacher_id": "3",
  "group_ids": [1],
  "invite_token": "<attendance_invite_jwt>",
  "join_url": "http://localhost:3000/#/attendance/join?token=<jwt>",
  "url": "http://localhost:3000/#/attendance/join?token=<jwt>",
  "qr_payload": "http://localhost:3000/#/attendance/join?token=<jwt>",
  "expires_minutes": 20,
  "expires_at": "2026-04-21T21:25:37+07:00",
  "schedule_start": "2026-04-21T21:15:24+07:00",
  "schedule_end": "2026-04-21T22:25:24+07:00",
  "roster_size": 25,
  "timezone": "Asia/Novosibirsk"
}
```

`qr_payload` рендерите в QR-код; `join_url` — та же ссылка для перехода.

**Ошибки:** `400`, `401`, `403`, `500`.

---

### GET `/api/teaching/attendance/session/active` — активная сессия

Возвращает не истёкшую сессию текущего преподавателя (для восстановления экрана после перезагрузки).

**Auth:** Bearer (teacher).

**Ответ 200 (`result`):**

```json
{
  "active": true,
  "remaining_seconds": 900,
  "seconds_remaining": 900,
  "server_time": "2026-04-21T21:10:37+07:00",
  "timezone": "Asia/Novosibirsk",
  "session": {
    "id": 5,
    "lesson_id": 5,
    "lesson_name": "Networks",
    "subject_id": 2,
    "teacher_id": 3,
    "is_active": true,
    "created_at": "2026-04-21T21:05:37+07:00",
    "expires_at": "2026-04-21T21:25:37+07:00",
    "remaining_seconds": 900,
    "marked_count": 18,
    "roster_size": 25,
    "attendance_percent": 72
  }
}
```

Если активной сессии нет — `active: false`, `session: null`.

**Ошибки:** `400`, `401`, `403`.

---

### GET `/api/teaching/attendance/session/timer` — таймер сессии

**Auth:** Bearer (teacher).

**Query:** `lesson_id` (int, обязателен) — ID сессии.

**Ответ 200 (`result`):**

```json
{
  "lesson_id": 5,
  "is_active": true,
  "remaining_seconds": 900,
  "seconds_remaining": 900,
  "expires_at": "2026-04-21T21:25:37+07:00",
  "server_time": "2026-04-21T21:10:37+07:00",
  "timezone": "Asia/Novosibirsk"
}
```

**Ошибки:** `400`, `401`, `403` (чужая сессия).

---

### GET `/api/teaching/attendance/session/marked-count` — сколько отметилось

**Auth:** Bearer (teacher).

**Query:** `lesson_id` (int, обязателен).

**Ответ 200 (`result`):**

```json
{
  "lesson_id": 5,
  "marked_count": 18,
  "roster_size": 25,
  "attendance_percent": 72
}
```

Удобно поллить вместе с таймером во время активной сессии.

**Ошибки:** `400`, `401`, `403`.

---

### GET `/api/teaching/attendance/session/roster` — живая таблица занятия

Возвращает полный состав текущей сессии с актуальным статусом каждого студента. Интерфейс преподавателя опрашивает endpoint раз в 2 секунды, поэтому самостоятельные отметки появляются в таблице без перезагрузки страницы.

**Auth:** Bearer (teacher).

**Query:** `lesson_id` (int, обязателен).

**Ответ 200 (`result`):**

```json
{
  "lesson_id": 5,
  "lesson_name": "Networks",
  "subject_id": 2,
  "server_time": "2026-04-21T21:12:06+07:00",
  "timezone": "Asia/Novosibirsk",
  "roster_size": 25,
  "marked_count": 18,
  "attendance_percent": 72,
  "students": [
    {
      "student_id": 4,
      "student_name": "Демин Сергей А.",
      "group_id": 237,
      "group_name": "ИКС-433",
      "status": "present",
      "marked_at": "2026-04-21T21:12:05+07:00",
      "marked_by": "self"
    }
  ]
}
```

`marked_by`: `self` для самостоятельной отметки студента, `teacher` для ручной правки преподавателя. У неотмеченного студента `status = absent`, а `marked_at` и `marked_by` могут отсутствовать.

**Ошибки:** `400`, `401`, `403` (чужая сессия).

---

### POST `/api/teaching/attendance/mark` — ручная правка посещения

Преподаватель вручную ставит статус студенту в своей сессии (поверх само-отметки). Ручная правка разрешена и после неудачной антифрод-проверки; факт срабатывания автопроверки сохраняется для аудита.

**Auth:** Bearer (teacher).

**Тело запроса:**

```json
{
  "lesson_id": 5,
  "student_id": 4,
  "status": "present"   // present | absent | late | excused
}
```

**Ответ 200:** стандартный конверт. **Ошибки:** `400`, `401`, `403`, `404`.

---

### GET `/api/teaching/subjects` — предметы преподавателя

Возвращает предметы и группы текущего преподавателя из расписания. Используйте для выпадающих списков.

**Auth:** Bearer (teacher).

**Ответ 200:** стандартный конверт, `result` — список предметов с группами.

**Ошибки:** `401`, `403`, `500`.

---

### POST `/api/teaching/attendance/group` — статистика посещаемости группы

**Auth:** Bearer (teacher).

**Тело запроса:**

```json
{
  "group_id": 1,
  "subject_id": 2   // опционально
}
```

**Ответ 200 (`result`):**

```json
{
  "group_id": 1,
  "subject_id": 2,
  "timezone": "Asia/Novosibirsk",
  "summary": { "sessions_count": 3, "students_count": 1 },
  "students": [
    {
      "student_id": 4,
      "student_name": "Test Student",
      "attended_sessions": 2,
      "excused_sessions": 0,
      "total_sessions": 3,
      "attendance_percent": 66.67,
      "last_marked_at": "2026-04-20T20:13:07+07:00"
    }
  ]
}
```

**Ошибки:** `400`, `401`, `403`, `500`.

---

### POST `/api/teaching/attendance/student/history` — история посещений студента

Детальная история студента по конкретному предмету.

**Auth:** Bearer (teacher).

**Тело запроса:**

```json
{
  "student_id": 4,
  "subject_id": 2
}
```

**Ответ 200 (`result`):**

```json
{
  "items": [
    { "date": "2026-04-20", "lesson_name": "Math", "status": "attended" }
  ]
}
```

**Ошибки:** `400`, `401`, `403`, `404`.

---

### POST `/api/teaching/group/performance` — сводка по группе (посещаемость + оценки)

Комбинированный отчёт по каждому студенту группы по предмету.

**Auth:** Bearer (teacher).

**Тело запроса:**

```json
{
  "group_id": 2,
  "subject_id": 1
}
```

**Ответ 200 (`result`):**

```json
{
  "group_id": 2,
  "group_name": "ИКС-433",
  "subject_id": 1,
  "subject_name": "Networks",
  "timezone": "Asia/Novosibirsk",
  "summary": {
    "students_count": 1,
    "sessions_count": 4,
    "avg_attendance_percent": 75,
    "avg_grade_percent": 65
  },
  "students": [
    {
      "student_id": 4,
      "student_name": "Test Student",
      "attended_sessions": 3,
      "excused_sessions": 0,
      "total_sessions": 4,
      "attendance_percent": 75,
      "current_score": 13,
      "passed_max": 10,
      "total_max": 20,
      "grade_percent": 65
    }
  ]
}
```

**Ошибки:** `400`, `401`, `403`, `404`.

---

## 4. Посещаемость — студент

### POST `/api/student/attendance/confirm` — подтвердить посещение по токену

Студент открывает `join_url` (или сканирует QR), фронт достаёт `token` из query и шлёт его сюда.

**Auth:** Bearer (student).

**Тело запроса:**

```json
{
  "invite_token": "<attendance_invite_jwt>",
  "device_id": "web-browser-uuid",
  "lat": 55.0084,
  "lon": 82.9357
}
```

`device_id`, `lat` и `lon` обязательны. Если пользователь запретил геолокацию или браузер не смог определить координаты, отметка не принимается. Повторное устройство в рамках одной сессии или удалённость более 200 м сохраняются как попытка нарушения и не засчитываются как посещение.

**Ответ 200 (`result`):**

```json
{
  "attendance": "confirmed",
  "lesson_id": "5",
  "student_id": "4",
  "teacher_id": "3",
  "subject_id": 2,
  "marked_at": "2026-04-20T20:13:07+07:00",
  "is_fraud": false
}
```

**Ошибки:** `400` (токен невалиден/просрочен), `401`, `403` (студент не из группы), `409` (уже отмечен), `500`.

---

### POST `/api/student/mark-attendance` — отметка с Android-клиента

Отметка с геолокацией и ID устройства (мобильное приложение).

**Auth:** Bearer (student).

**Тело запроса:**

```json
{
  "invite_token": "<attendance_invite_jwt>",
  "lesson_id": 5,
  "device_id": "android-device-uuid",
  "lat": 55.0084,
  "lon": 82.9357
}
```

**Ответ 200:** стандартный конверт. **Ошибки:** `400`, `401`, `403`, `409` (уже отмечен / устройство уже использовалось).

---

### GET `/api/student/attendance/history` — история посещений (календарь)

Отметки текущего студента, сгруппированные по датам, за выбранный год. Подходит для «теплокарты» активности.

**Auth:** Bearer (student).

**Query:** `year` (int, по умолчанию текущий год).

**Ответ 200 (`result`):**

```json
{
  "year": 2026,
  "items": [
    { "date": "2026-04-20", "count": 1 }
  ]
}
```

**Ошибки:** `400`, `401`, `403`, `500`.

---

## 5. Оценки — преподаватель

### POST `/api/teaching/grades/items` — создать контрольную точку

**Auth:** Bearer (teacher).

**Тело запроса:**

```json
{
  "subject_id": 1,
  "title": "Лабораторная 1",
  "item_type": "lab",
  "max_score": 10,
  "deadline": "2026-05-01"
}
```

**Ответ 200:** стандартный конверт (созданный item в `result`). **Ошибки:** `400`, `401`, `403`, `404`.

---

### POST `/api/teaching/grades/items/list` — список контрольных точек предмета

**Auth:** Bearer (teacher).

**Тело запроса:**

```json
{ "subject_id": 1 }
```

**Ответ 200:** стандартный конверт, `result` — массив контрольных точек.

**Ошибки:** `400`, `401`, `403`, `404`.

---

### POST `/api/teaching/grades` — поставить/обновить оценку

Upsert: если оценка по этой точке у студента уже есть — обновляется.

**Auth:** Bearer (teacher).

**Тело запроса:**

```json
{
  "item_id": 7,
  "student_id": 4,
  "score": 8,
  "comment": "Хорошо",       // опционально
  "session_id": 5            // опционально
}
```

**Ответ 200:** стандартный конверт. **Ошибки:** `400` (score > max_score и т.п.), `401`, `403`, `404`.

---

### POST `/api/teaching/grades/student` — ведомость студента по предмету

**Auth:** Bearer (teacher).

**Тело запроса:**

```json
{
  "student_id": 4,
  "subject_id": 1
}
```

**Ответ 200:** стандартный конверт, `result` — ведомость (контрольные точки с баллами студента и итогами).

**Ошибки:** `400`, `401`, `403`, `404`.

---

### POST `/api/teaching/student/performance/radar` — радар успеваемости студента

Точка на радаре по каждому предмету студента, которого ведёт преподаватель.

**Auth:** Bearer (teacher).

**Тело запроса:**

```json
{ "student_id": 4 }
```

**Ответ 200:** стандартный конверт, `result` — массив `{предмет, процент}` для радар-чарта.

**Ошибки:** `400`, `401`, `403`, `404`.

---

## 6. Оценки — студент

### POST `/api/student/grades` — свои оценки по предмету

**Auth:** Bearer (student).

**Тело запроса:**

```json
{ "subject_id": 1 }
```

**Ответ 200:** стандартный конверт, `result` — ведомость студента по предмету.

**Ошибки:** `400`, `401`, `403`, `404`.

---

### GET `/api/student/grades/all` — все оценки одним запросом

Все предметы учебного плана с контрольными точками, суммами по предмету,
процентом посещаемости по каждому предмету и общим итогом. Для посещаемости
возвращаются `total_sessions`, `attended_sessions`, `excused_sessions`,
`missed_sessions` и `attendance_percent`; уважительные пропуски исключаются из
знаменателя процента. Подходит для главного экрана «Оценки».

**Auth:** Bearer (student).

**Ответ 200:** стандартный конверт. **Ошибки:** `400`, `401`, `403`, `404`, `500`.

---

### GET `/api/student/performance/radar` — радар успеваемости

По одной точке на предмет (из расписания группы) с процентом набранных баллов за уже оценённые работы семестра.

**Auth:** Bearer (student).

**Ответ 200:** стандартный конверт. **Ошибки:** `401`, `403`, `500`.

---

## 7. Посещаемость и расписание

### GET `/api/student/attendance/summary` — процент посещаемости

Сводная посещаемость текущего студента за семестр и разбивка по предметам.

**Auth:** Bearer (student).

**Query:** `semester_id` (int, необязательно; по умолчанию открытый семестр).

**Ответ 200:** стандартный конверт; `result.summary` содержит общий процент,
`result.subjects` — процент и счётчики по каждому предмету.

**Ошибки:** `400`, `401`, `403`, `404`, `500`.

---

### GET `/api/student/schedule/day` — расписание студента на день

Пары текущего студента на дату. Доступна только текущая и следующая календарная неделя.

**Auth:** Bearer (student).

**Query:** `date` (string, `YYYY-MM-DD`, по умолчанию сегодня).

**Ответ 200:** стандартный конверт, `result` — список пар на день.

**Ошибки:** `400` (дата вне разрешённого диапазона / кривой формат), `401`, `403`, `500`.

---

### GET `/api/teaching/schedule/day` — расписание преподавателя на день

Пары текущего преподавателя на дату с предметом, группой, типом занятия,
аудиторией и подгруппой. Доступна только текущая и следующая календарная неделя.

**Auth:** Bearer (teacher).

**Query:** `date` (string, `YYYY-MM-DD`, по умолчанию сегодня).

**Ответ 200:** стандартный конверт, `result.lessons` — список пар с полями
`schedule_id`, `lesson_num`, `start_time`, `end_time`, `subject_id`,
`subject_name`, `group_id`, `group_name`, `lesson_type`, `room_info`, `subgroup`.

**Ошибки:** `400`, `401`, `403`, `500`.

---

## 8. Персонал (head / dean / admin)

### GET `/api/staff/overview` — обзор по зоне ответственности

Группы, преподаватели и студенты в рамках роли вызывающего:
- `teacher` → свои группы;
- `head` → своя кафедра;
- `dean` → свой факультет;
- `admin` → всё.

**Auth:** Bearer.

**Ответ 200:** стандартный конверт. **Ошибки:** `401`, `403`.

---

### GET `/api/student/ratings/group` — рейтинг своей группы

Возвращает ту же структуру общего рейтинга, но только для группы авторизованного
студента. Чужую группу выбрать параметром запроса нельзя. Для студентов с
действующим согласием возвращается ФИО, без согласия — стабильный код `STU-...`.
Поле `is_current_user` отмечает строку авторизованного студента.

**Auth:** Bearer (student). Необязательный query-параметр `semester_id`; без него
используется открытый семестр.

**Ответ 200:** стандартный конверт; объект схемы `1.0` находится в поле `result`.
**Ошибки:** `400`, `401`, `404`, `500`.

---

### GET `/api/staff/ratings/general` — данные общего рейтинга

Подробная JSON-выгрузка за семестр: кафедры, предметы и преподаватели, группы,
студенты, статус согласия на обработку персональных данных, посещаемость,
лабораторные/практические работы и сводные проценты по каждому предмету.

**Auth:** Bearer (teacher и руководящие роли). Данные ограничены зоной видимости
роли. Необязательный query-параметр `semester_id`; без него используется открытый
семестр.

**Ответ 200:** стандартный конверт; объект схемы `1.0` находится в поле `result`.
**Ошибки:** `400`, `401`, `403`, `404`, `500`.

---

### GET `/api/staff/analytics` — интерактивная аналитика

Возвращает единый срез аналитики в зоне видимости пользователя: общий рейтинг,
посещаемость, медиану и разброс, покрытие оценками, риски, распределение,
предметную heatmap, структуру посещений и недельную динамику. Доступные уровни:
`faculty`, `stream`, `group`, `student`. Поток определяется полем `groups.stream_name`;
для групп без значения используется поток `__none__` / «Без потока».

**Auth:** Bearer (teacher и руководящие роли). Данные ограничены зоной роли.

**Query:** `semester_id` (необязательный), `scope_type` (по умолчанию `faculty`),
`scope_id` (обязателен для потока/группы/студента), `subject_id` (необязательный).

В `result.weekly` каждая точка содержит `week_start`, `week_end`,
`average_rating`, `median_rating`, `attendance_percent`, `grade_coverage` и
количество студентов с данными. Рейтинг считается snapshot-ом на конец недели:
учитываются только уже наступившие контрольные точки и оценки, существовавшие
на этот момент. Уважительные причины исключаются из знаменателя посещаемости,
опоздание считается присутствием.

**Ответ 200:** стандартный конверт; данные фильтров находятся в
`result.options`, сводка — в `result.summary`, агрегаты — в `groups`, `streams`,
`students`, `subjects`.
**Ошибки:** `400`, `401`, `403`, `404`, `500`.

---

### GET `/api/staff/reports/performance.xlsx` — отчёт Excel

Рейтинговый отчёт: один лист по всей зоне + по листу на группу. Студенты отсортированы по рейтингу, с процентами по предметам и посещаемостью.

**Auth:** Bearer (head / dean / admin).

**Ответ 200:** бинарный файл `.xlsx` (`Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`). На фронте скачивайте как blob.

**Ошибки:** `401`, `403`, `500`.

---

### GET `/api/staff/reports/performance.pdf` — отчёт PDF

То же, что xlsx, но PDF с цветовой кодировкой процентов.

**Auth:** Bearer (head / dean / admin).

**Ответ 200:** бинарный файл `application/pdf`. **Ошибки:** `401`, `403`, `500`.

---

## 9. Администратор — пользователи

Все методы раздела доступны только роли `admin`.

### GET `/api/admin/users` — список пользователей

Параметры: `page`, `page_size`, `search`, `role`, `status`.

Допустимые статусы: `active`, `blocked`, `archived`. Поиск работает по логину и email.

### GET `/api/admin/users/:user_id` — один пользователь

Возвращает логин, основную роль (`role`), все назначенные роли (`roles`), email, статус, привязанные профили/области доступа, состояние 2FA и даты создания/обновления.

### POST `/api/admin/users` — создать пользователя

Общие поля:

```json
{
  "login": "dean_teacher_10",
  "password": "strong_password",
  "roles": ["dean", "teacher", "admin"],
  "primary_role": "dean",
  "email": "teacher@example.com",
  "full_name": "Иванов Иван Иванович",
  "faculty_id": 1,
  "job_title": "Преподаватель"
}
```

Для обратной совместимости вместо `roles` и `primary_role` можно передать одно поле `role`. Для `student` передаются `full_name` и необязательный `group_id`. Для `teacher` — `full_name`, необязательные `lectern_id` и `job_title`. Для `head`, `secretary`, `program_creator` обязателен `lectern_id`; для `dean`, `director` — `faculty_id`.

Пользователь и его профиль создаются одной транзакцией.

### PATCH `/api/admin/users/:user_id` — изменить пользователя

Можно передать только изменяемые поля: `login`, `password`, `email`, `roles`, `primary_role`, `status`. Легаси-поле `role` заменяет весь набор ролей одной ролью.

При добавлении `student` требуется `student_id`, при добавлении `teacher` — `teacher_id`; для кафедральных и факультетских ролей передаются `lectern_id` и `faculty_id` соответственно.

Администратор не может изменить собственные роли или статус. Последнего активного администратора нельзя заблокировать или лишить роли. Изменение ролей отзывает старые токены пользователя.

### DELETE `/api/admin/users/:user_id` — архивировать пользователя

Физическое удаление не выполняется. Пользователь получает статус `archived`, после чего его токены перестают приниматься.

---

## 10. Прочее / Android

### POST `/lessons/create` — создать пару (Android-клиент)

Преподаватель создаёт занятие по именам или ID предмета/групп и текущей геолокации.

**Auth:** Bearer (teacher).

**Тело запроса:**

```json
{
  "lesson_name": "Networks",
  "subject": "Сети",          // или subject_id
  "subject_id": 2,
  "groups": ["ИКС-433"],      // или group_ids
  "group_ids": [1],
  "teacher_id": 3,
  "expires_minutes": 20,
  "lat": 55.0084,
  "lon": 82.9357
}
```

**Ответ 200:** стандартный конверт. **Ошибки:** `400`, `401`, `403`.

---

### GET `/uploads/*` — статика

Публичная раздача загруженных файлов (аватары и т.п.). Auth не требуется.

---

## Сводная таблица

| Метод | Путь | Роль | Назначение |
|---|---|---|---|
| POST | `/register` | — | Регистрация (invite_code / легаси role_hash) |
| POST | `/register/by-invite` | — | Регистрация по инвайт-коду |
| POST | `/login` | — | Вход, выдача JWT |
| POST | `/api/auth/switch-role` | любая назначенная | Переключить активную роль и получить новый JWT |
| GET | `/profile` | любая | Профиль текущего пользователя |
| POST | `/api/auth/forgot-password` | — | Запрос сброса пароля |
| POST | `/api/auth/reset-password` | — | Сброс пароля по токену |
| POST | `/api/user/email` | любая | Привязка email |
| POST | `/api/user/upload-avatar` | любая | Загрузка аватара (multipart) |
| POST | `/api/teaching/attendance-link` | teacher, head | Создать сессию посещаемости (+QR) |
| POST | `/api/teaching/attendance/session` | teacher, head | Алиас предыдущей |
| GET | `/api/teaching/attendance/session/active` | teacher, head | Активная сессия |
| GET | `/api/teaching/attendance/session/timer` | teacher, head | Таймер сессии |
| GET | `/api/teaching/attendance/session/marked-count` | teacher, head | Счётчик отметившихся |
| GET | `/api/teaching/attendance/session/roster` | teacher, head | Живая таблица отметок по студентам |
| POST | `/api/teaching/attendance/mark` | teacher, head | Ручная правка статуса |
| GET | `/api/teaching/subjects` | teacher, head | Предметы и группы преподавателя |
| POST | `/api/teaching/attendance/group` | teacher, head | Посещаемость по группе |
| POST | `/api/teaching/attendance/student/history` | teacher, head | История посещений студента |
| POST | `/api/teaching/group/performance` | teacher, head | Сводка группа+предмет |
| POST | `/api/student/attendance/confirm` | student | Подтвердить посещение по токену |
| POST | `/api/student/mark-attendance` | student | Отметка с Android (гео + device) |
| GET | `/api/student/attendance/history` | student | История посещений по датам |
| GET | `/api/student/attendance/summary` | student | Процент посещаемости по семестру и предметам |
| POST | `/api/teaching/grades/items` | teacher, head | Создать контрольную точку |
| POST | `/api/teaching/grades/items/list` | teacher, head | Список контрольных точек |
| POST | `/api/teaching/grades` | teacher, head | Поставить/обновить оценку |
| POST | `/api/teaching/grades/student` | teacher, head | Ведомость студента |
| POST | `/api/teaching/student/performance/radar` | teacher, head | Радар студента |
| POST | `/api/student/grades` | student | Свои оценки по предмету |
| GET | `/api/student/grades/all` | student | Все оценки разом |
| GET | `/api/student/performance/radar` | student | Свой радар успеваемости |
| GET | `/api/student/schedule/day` | student | Расписание на день |
| GET | `/api/teaching/schedule/day` | teacher, head | Расписание преподавателя на день |
| GET | `/api/admin/users` | admin | Список пользователей |
| GET | `/api/admin/users/:user_id` | admin | Получить пользователя |
| POST | `/api/admin/users` | admin | Создать пользователя, набор ролей и профили |
| PATCH | `/api/admin/users/:user_id` | admin | Изменить пользователя, роли, основную роль или статус |
| DELETE | `/api/admin/users/:user_id` | admin | Архивировать пользователя |
| GET | `/api/staff/overview` | teacher+ | Обзор групп/людей по роли |
| GET | `/api/staff/analytics` | teacher+ | Рейтинг, посещаемость и динамика по зоне роли |
| GET | `/api/staff/reports/performance.xlsx` | head/dean/admin | Excel-отчёт |
| GET | `/api/staff/reports/performance.pdf` | head/dean/admin | PDF-отчёт |
| POST | `/lessons/create` | teacher | Создать пару (Android) |
| GET | `/uploads/*` | — | Статика (аватары) |
| GET | `/swagger/*` | — | Swagger UI |
