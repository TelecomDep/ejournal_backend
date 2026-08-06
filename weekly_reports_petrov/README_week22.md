# Неделя 22

## Часть 1. Отвязка электронной почты от профиля

На этой неделе была продолжена работа с настройками профиля пользователя. Раньше backend позволял привязать или изменить электронную почту, но отдельного способа удалить уже сохранённый адрес не было.

Добавлен новый маршрут:

- `DELETE /api/user/email` — удалить электронную почту текущего пользователя.

Пользователь определяется по JWT-токену из заголовка `Authorization`. Передавать `user_id` или сам адрес почты в запросе не требуется. Это не позволяет одному пользователю удалить почту другого пользователя.

В базе данных запись пользователя не удаляется. Поле `email` переводится в состояние `NULL`:

```go
func (r *UserRepository) DeleteEmail(ctx context.Context, userID int32) error {
    if userID <= 0 {
        return fmt.Errorf("invalid userID")
    }

    _, err := r.pool.Exec(
        ctx,
        `UPDATE users SET email = NULL WHERE id = $1`,
        userID,
    )

    if err != nil {
        return fmt.Errorf("delete email: %w", err)
    }

    return nil
}
```

После удаления пользователь может повторно привязать тот же или другой адрес через существующий процесс подтверждения по коду.

## Часть 2. Базовая система уведомлений

Для раздела уведомлений на frontend была добавлена отдельная backend-модель. Уведомления разделены на четыре категории:

- `grades` — новые и изменённые оценки;
- `schedule` — создание, перенос и отмена занятий;
- `attendance` — открытие отметки и результат проверки посещаемости;
- `system` — системные предупреждения и сообщения администратора.

Конкретное событие хранится отдельно в поле `event_type`. Например, для категории `grades` используются события `grade_created` и `grade_updated`, а для категории `system` — `fraud` и `admin_update`.

Добавлены три таблицы:

```sql
CREATE TABLE notifications (
    notification_id BIGSERIAL PRIMARY KEY,
    category VARCHAR(32) NOT NULL,
    event_type VARCHAR(64) NOT NULL,
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    created_by_user_id INTEGER REFERENCES users(id),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE notification_recipients (
    notification_id BIGINT REFERENCES notifications(notification_id),
    user_id INTEGER REFERENCES users(id),
    read_at TIMESTAMPTZ,
    PRIMARY KEY (notification_id, user_id)
);

CREATE TABLE notification_settings (
    user_id INTEGER PRIMARY KEY REFERENCES users(id),
    grades BOOLEAN NOT NULL DEFAULT TRUE,
    schedule BOOLEAN NOT NULL DEFAULT TRUE,
    attendance BOOLEAN NOT NULL DEFAULT TRUE,
    system BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Таблица `notification_recipients` отделена от самих уведомлений, потому что одно сообщение может быть отправлено сразу нескольким пользователям. При этом каждый получатель имеет собственное время прочтения.

В поле `metadata` сохраняются дополнительные данные события: идентификатор занятия, студента, оценки, контрольной точки или причина обнаруженного фрауда.

## Часть 3. Пользовательские маршруты уведомлений

Для frontend добавлены маршруты получения уведомлений и управления их состоянием:

- `GET /api/user/notifications` — список уведомлений текущего пользователя;
- `GET /api/user/notifications/unread-count` — количество непрочитанных уведомлений;
- `PATCH /api/user/notifications/:notification_id/read` — отметить одно уведомление прочитанным;
- `PATCH /api/user/notifications/read-all` — отметить прочитанными все уведомления;
- `GET /api/user/notification-settings` — получить настройки категорий;
- `PUT /api/user/notification-settings` — сохранить настройки категорий.

Список поддерживает пагинацию, фильтрацию по категории и выдачу только непрочитанных записей:

```text
GET /api/user/notifications?page=1&page_size=20
GET /api/user/notifications?category=grades
GET /api/user/notifications?unread_only=true
```

Пример элемента в ответе:

```json
{
  "notification_id": 15,
  "category": "system",
  "event_type": "fraud",
  "title": "Обнаружена подозрительная отметка",
  "message": "Система отклонила отметку посещаемости.",
  "metadata": {
    "lesson_id": 42,
    "student_id": 10,
    "fraud_reason": "device_id already used in this lesson"
  },
  "is_read": false,
  "created_at": "2026-08-06T08:30:00Z"
}
```

При отметке уведомления прочитанным backend обязательно проверяет одновременно `notification_id` и `user_id`. Поэтому пользователь не может изменить состояние чужого уведомления.

## Часть 4. Автоматическое создание уведомлений

Система уведомлений подключена к существующим операциям оценок и посещаемости.

После сохранения оценки студент получает одно из событий:

- `grade_created` — преподаватель выставил новую оценку;
- `grade_updated` — преподаватель изменил существующую оценку.

В уведомлении передаются `grade_id`, `student_id`, `item_id`, `subject_id`, текущий балл и максимальный балл контрольной точки.

При создании сессии посещаемости студенты выбранных групп получают событие `attendance_opened`. Уведомление имеет время окончания, совпадающее со временем действия ссылки отметки.

После попытки отметить посещаемость создаётся `attendance_marked` или `attendance_rejected`. Если система обнаружила слишком большое расстояние до занятия или повторное использование устройства, дополнительно создаётся системное событие `fraud`.

Фрауд-уведомление получают:

- студент, чья отметка была отклонена;
- преподаватель, который проводит занятие;
- активные администраторы.

Причина сохраняется в `metadata`, поэтому frontend может показать пользователю краткий текст, а администратору — более подробную информацию.

## Часть 5. Административные уведомления и аудитория

Администратору добавлены отдельные маршруты:

- `GET /api/admin/notifications` — список созданных уведомлений;
- `POST /api/admin/notifications` — создание уведомления;
- `PATCH /api/admin/notifications/:notification_id` — изменение текста и срока действия;
- `DELETE /api/admin/notifications/:notification_id` — удаление уведомления.

Обычный пользователь не имеет доступа к этим маршрутам. Роль проверяется по текущей учётной записи, загруженной из базы данных.

При создании сообщения администратор может выбрать аудиторию:

- `all` — все активные пользователи;
- `role` — пользователи выбранной роли;
- `users` — конкретный список `user_ids`;
- `groups` — студенты выбранных учебных групп.

Пример сообщения для всех пользователей:

```json
{
  "category": "system",
  "event_type": "admin_update",
  "title": "Обновление электронного журнала",
  "message": "Добавлен новый раздел аналитики успеваемости.",
  "audience": "all"
}
```

Для сообщения об изменении расписания администратор может указать категорию `schedule`, событие `lesson_rescheduled` и отправить его только нужным группам.

## Часть 6. Обновление Swagger

Для всех пользовательских и административных маршрутов уведомлений добавлены Swagger-аннотации. Повторно сгенерированы файлы:

- `backend/docs/docs.go`;
- `backend/docs/swagger.json`;
- `backend/docs/swagger.yaml`.

Также исправлено старое описание маршрута получения аватара. Раньше оно ошибочно повторяло аннотацию загрузки изображения, из-за чего генератор сообщал о дублирующемся `POST /api/user/upload-avatar`. Теперь получение изображения правильно описано как `GET /api/user/avatar/{user_id}`.

В результате frontend получил документированный контракт для загрузки истории уведомлений, счётчика непрочитанных записей, настроек категорий и административной отправки сообщений. Backend успешно собирается после добавленных изменений.
