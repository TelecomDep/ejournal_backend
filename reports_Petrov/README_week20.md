# Неделя 20

## Часть 1. Backend: единая модель учебных семестров

На этой неделе основной задачей стала полноценная привязка учебных данных к семестрам. Раньше оценки и посещаемость не имели явной связи с учебным периодом: при переходе на новый семестр старые и новые данные могли попадать в одну сводку, а ограничение БРС в 100 баллов действовало сразу на весь предмет.

Добавлена отдельная сущность `semesters`:

```sql
CREATE TABLE IF NOT EXISTS semesters (
    semester_id SERIAL PRIMARY KEY,
    academic_year VARCHAR(20) NOT NULL,
    term_num SMALLINT NOT NULL CHECK (term_num IN (1, 2)),
    name VARCHAR(255) NOT NULL,
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    is_current BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT semesters_year_term_unique UNIQUE (academic_year, term_num),
    CONSTRAINT semesters_range_check CHECK (ends_at > starts_at)
);
```

Для семестра хранятся учебный год, номер половины учебного года, отображаемое название, даты начала и окончания, а также признак активного семестра. Частичный уникальный индекс гарантирует, что активным может быть только один семестр:

```sql
CREATE UNIQUE INDEX IF NOT EXISTS idx_semesters_current
    ON semesters (is_current)
    WHERE is_current;
```

В `attendance_sessions`, `grade_items`, `semester_load` и `subject_controls` добавлен внешний ключ `semester_id`. Для старых оценок и занятий миграция выполняет обратную привязку по дате, а если подходящий период не найден — использует активный семестр.

## Часть 2. Репозиторий и управление активным семестром

Добавлен `SemesterRepository`, который отвечает за создание, получение, список и активацию семестров. Переключение активного периода выполняется внутри одной транзакции: сначала снимается предыдущий признак `is_current`, затем включается выбранный семестр.

```go
func (r *SemesterRepository) Activate(ctx context.Context, id int32) (Semester, bool, error) {
    tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
    if err != nil {
        return Semester{}, false, fmt.Errorf("begin activate semester transaction: %w", err)
    }
    defer func() { _ = tx.Rollback(ctx) }()

    if _, err := tx.Exec(ctx,
        `UPDATE semesters SET is_current = FALSE WHERE is_current = TRUE`,
    ); err != nil {
        return Semester{}, false, err
    }

    semester, err := scanSemester(tx.QueryRow(ctx, `
        UPDATE semesters
        SET is_current = TRUE
        WHERE semester_id = $1
        RETURNING semester_id, academic_year, term_num, name,
                  starts_at, ends_at, is_current, created_at`,
        id,
    ))
    // ...
}
```

На сервисном уровне добавлена единая функция разрешения семестра. Если клиент передает `semester_id`, используется выбранный период; если поле не передано, автоматически берется текущий семестр. Это сохраняет совместимость со старыми клиентами, которые еще не умеют выбирать учебный период.

## Часть 3. API семестров

Добавлены четыре backend-маршрута:

- `GET /api/semesters` — список всех семестров;
- `GET /api/semesters/current` — текущий активный семестр;
- `POST /api/admin/semesters` — создание семестра администратором;
- `PATCH /api/admin/semesters/:semester_id/activate` — переключение активного семестра.


## Часть 4. Разделение оценок и БРС по семестрам

Контрольные точки теперь создаются с `semester_id`, а все основные выборки оценок фильтруются по предмету и семестру. Выбранный период учитывается в ведомости студента, общей выдаче оценок, диаграмме успеваемости и сводке преподавателя.

Ограничение БРС в 100 баллов также стало семестровым:

```sql
SELECT COALESCE(SUM(max_score), 0)
INTO total_max
FROM public.grade_items
WHERE subject_id = NEW.subject_id
  AND semester_id = NEW.semester_id
  AND deleted_at IS NULL
  AND item_id IS DISTINCT FROM NEW.item_id;
```

Теперь один предмет может иметь отдельный набор контрольных точек до 100 баллов в каждом семестре, а архивные оценки не смешиваются с текущей успеваемостью.

## Часть 5. Разделение посещаемости и отчетов

Каждая новая сессия посещаемости сохраняет семестр, в котором она была создана. Это работает как для обычного преподавательского API, так и для Android-ручки с геолокацией.

Семестр учитывается в:

- статистике посещаемости группы;
- объединенной сводке посещаемости и успеваемости;
- активной сессии преподавателя;
- служебном отчете по успеваемости;
- PDF- и Excel-модели отчета.

В ответы API добавлены `semester_id` и описание семестра, чтобы клиент мог показать пользователю, за какой учебный период рассчитаны показатели.
