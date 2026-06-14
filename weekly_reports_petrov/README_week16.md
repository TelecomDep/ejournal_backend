# Неделя 16 (14 июня 2026)

## Диаграмма успеваемости студента (radar chart): новый backend API + миграция демо-аккаунта преподавателя

### 1. Цель дня

Две задачи на бэкенде:

- починить демо-аккаунт преподавателя (`teacher_test`), у которого Android-клиент видел только одну группу, хотя в БД их больше;
- добавить API для диаграммы успеваемости студента за семестр (по предметам из расписания группы), которую может смотреть и сам студент, и преподаватель — по любому своему студенту.

### 2. Миграция: у `teacher_test` теперь несколько групп

Ранее демо-преподаватель был привязан ровно к одной строке `schedules` → одной группе, поэтому `teacher_subjects` (он строится по расписанию преподавателя) всегда отдавал одну группу, даже если в `groups` их много.

Новая миграция `20260614090000_demo_teacher_multi_group.sql` пересобирает «демо-занятие» (`lesson_num = 99`) так, чтобы оно покрывало **все группы, в которых есть хотя бы один студент** (фолбэк — первые 5 групп, если студентов вообще нет). День/тип недели для `schedules` вычисляется в формате «индекса парсера» (0–13), который затем нормализует существующий триггер `normalize_schedule_day_idx`.

Миграция применена через `docker compose up migrate` и проверена `psql` — `goose` отрапортовал `successfully migrated database to version: 20260614090000`.

### 3. Новый backend для диаграммы успеваемости

Метрика по оси (предмету): **набрано баллов / максимум баллов** среди работ, у которых `deadline < now()` (т.е. уже «пройденные» по сроку). Оси — все предметы из расписания группы студента за семестр, даже если по ним ещё нет оценок (тогда 0%).

Слои реализации:

1. **`internal/db/models.go`** — новая структура `SubjectPerformancePoint` (одна точка/ось диаграммы).
2. **`internal/db/grade_repository.go`** — `GetStudentPerformanceRadar`: один SQL-запрос, который берёт предметы из расписания группы студента и агрегирует баллы только по работам с прошедшим дедлайном.
3. **`internal/app/grades.go`**:
   - `TeacherStudentRadarData` — payload для ручки преподавателя;
   - `teacherCanViewStudent` — проверка, что преподаватель ведёт группу этого студента (или это `teacher_test`);
   - `performanceRadarResult` — общая сборка ответа;
   - `studentPerformanceRadar` — версия для студента (своя диаграмма);
   - `teacherStudentPerformanceRadar` — версия для преподавателя (диаграмма любого своего студента), с проверкой роли, существования студента и доступа;
   - в `GradeHTTPStatus` добавлена новая 403-ветка `"forbidden: teacher does not teach this student"`.
4. **`internal/app/service.go`** — два новых action-кейса в `handleRequest`: `student_performance_radar` и `teacher_student_performance_radar`.
5. **`internal/httpserver/server.go`** — два новых маршрута со Swagger-аннотациями:
   - `GET /api/student/performance/radar`
   - `POST /api/teacher/student/performance/radar`

### 4. Регенерация Swagger

После добавления новых хендлеров и схемы `TeacherStudentRadarData` пересобрана документация:

```bash
swag init -g cmd/server/main.go -o docs
```

Обновились `docs/docs.go`, `docs/swagger.json`, `docs/swagger.yaml` — оба новых пути и схема запроса попали в спецификацию.

### 5. Тестирование и валидация (docker-compose)

- `docker compose up migrate --abort-on-container-exit` — новая миграция применена чисто, проверена через `docker exec ejournal-postgres psql`.
- `docker compose up -d --build ejournal-backend` — бэкенд пересобран и поднят на `:8888`.
- `go build ./...`, `gofmt -w` — без ошибок.
- Ручная проверка через `curl` под `teacher_test` / `student_test` (пароль `123456`):
  - `GET /api/student/performance/radar` → `200`, `subject_id:1 "Networks" score:5 max_score:10 percent:50`;
  - `POST /api/teacher/student/performance/radar {"student_id":2}` → `200`, тот же результат;
  - без токена → `401`; студент на ручке преподавателя и наоборот → `403`;
  - `student_id: 0` → `400 student_id is required`; `student_id: 99999` → `404 student not found`.

### 6. Итог дня

Демо-преподаватель теперь видит несколько групп, как и должно быть. Добавлен полностью рабочий и проверенный backend для диаграммы успеваемости — для студента (своя диаграмма) и для преподавателя (диаграмма по любому студенту, которого он ведёт), с корректными кодами ошибок на все нештатные случаи. Swagger-документация синхронизирована с новыми ручками.

### 7. Ключевые вставки кода (backend)

#### 7.1 Модель точки диаграммы

```go
type SubjectPerformancePoint struct {
	SubjectID   int32  `json:"subject_id"`
	SubjectName string `json:"subject_name"`
	Score       int32  `json:"score"`
	MaxScore    int32  `json:"max_score"`
	Percent     int32  `json:"percent"`
}
```

#### 7.2 Запрос к БД: предметы из расписания группы + баллы по прошедшим дедлайнам

```go
func (r *GradeRepository) GetStudentPerformanceRadar(ctx context.Context, studentID int32) ([]SubjectPerformancePoint, error) {
	if studentID <= 0 {
		return nil, fmt.Errorf("student id is required")
	}

	rows, err := r.pool.Query(
		ctx,
		`WITH student_subjects AS (
		     SELECT DISTINCT sch.subject_id
		     FROM students st
		     JOIN schedules sch ON sch.group_id = st.group_id
		     WHERE st.student_id = $1
		 )
		 SELECT sub.subject_id,
		        sub.name,
		        COALESCE(SUM(g.score)      FILTER (WHERE gi.deadline < now()), 0)::INTEGER AS passed_score,
		        COALESCE(SUM(gi.max_score) FILTER (WHERE gi.deadline < now()), 0)::INTEGER AS passed_max
		 FROM student_subjects ss
		 JOIN subjects sub ON sub.subject_id = ss.subject_id
		 LEFT JOIN grade_items gi ON gi.subject_id = sub.subject_id
		 LEFT JOIN grades g ON g.item_id = gi.item_id AND g.student_id = $1
		 GROUP BY sub.subject_id, sub.name
		 ORDER BY sub.name, sub.subject_id`,
		studentID,
	)
	if err != nil {
		return nil, fmt.Errorf("get student performance radar: %w", err)
	}
	defer rows.Close()

	result := make([]SubjectPerformancePoint, 0)
	for rows.Next() {
		var point SubjectPerformancePoint
		if err := rows.Scan(&point.SubjectID, &point.SubjectName, &point.Score, &point.MaxScore); err != nil {
			return nil, fmt.Errorf("scan student performance radar row: %w", err)
		}
		if point.MaxScore > 0 {
			percent := int32((float64(point.Score) / float64(point.MaxScore)) * 100)
			if percent < 0 {
				percent = 0
			}
			if percent > 100 {
				percent = 100
			}
			point.Percent = percent
		}
		result = append(result, point)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate student performance radar rows: %w", err)
	}

	return result, nil
}
```

#### 7.3 Проверка доступа преподавателя к студенту

```go
func (s *Service) teacherCanViewStudent(ctx context.Context, teacherID, studentID int32) (bool, error) {
	var allowed bool
	err := s.store.Pool().QueryRow(
		ctx,
		`SELECT EXISTS (
		     SELECT 1
		     FROM schedules sch
		     JOIN students st ON st.group_id = sch.group_id
		     WHERE sch.teacher_id = $1 AND st.student_id = $2
		 ) OR EXISTS (
		     SELECT 1
		     FROM teachers t
		     JOIN users u ON u.id = t.user_id
		     WHERE t.teacher_id = $1
		       AND u.login = 'teacher_test'
		 )`,
		teacherID,
		studentID,
	).Scan(&allowed)
	if err != nil {
		return false, fmt.Errorf("check teacher student access: %w", err)
	}
	return allowed, nil
}
```

#### 7.4 Сервисные функции для студента и преподавателя

```go
type TeacherStudentRadarData struct {
	StudentID int32 `json:"student_id"`
}

func (s *Service) performanceRadarResult(studentID int32) Response {
	ctx, cancel := s.dbContext()
	defer cancel()

	points, err := s.store.Grades.GetStudentPerformanceRadar(ctx, studentID)
	if err != nil {
		return Response{OK: false, Error: "failed to load performance radar"}
	}

	return Response{
		OK: true,
		Result: map[string]any{
			"student_id": studentID,
			"subjects":   points,
		},
	}
}

func (s *Service) studentPerformanceRadar(sessionToken string) Response {
	studentUser, err := s.userBySessionToken(sessionToken)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if studentUser.Role != "student" {
		return Response{OK: false, Error: "forbidden: student role required"}
	}

	studentProfile, err := s.studentProfileByUser(studentUser)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	return s.performanceRadarResult(studentProfile.ID)
}

func (s *Service) teacherStudentPerformanceRadar(sessionToken string, data TeacherStudentRadarData) Response {
	teacherUser, err := s.userBySessionToken(sessionToken)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if teacherUser.Role != "teacher" {
		return Response{OK: false, Error: "forbidden: teacher role required"}
	}
	if data.StudentID <= 0 {
		return Response{OK: false, Error: "student_id is required"}
	}

	teacherProfile, err := s.teacherProfileByUser(teacherUser)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	_, found, err := s.store.Students.GetByID(ctx, data.StudentID)
	if err != nil {
		return Response{OK: false, Error: "failed to load student"}
	}
	if !found {
		return Response{OK: false, Error: "student not found"}
	}

	allowed, err := s.teacherCanViewStudent(ctx, teacherProfile.ID, data.StudentID)
	if err != nil {
		return Response{OK: false, Error: "failed to check teacher student access"}
	}
	if !allowed {
		return Response{OK: false, Error: "forbidden: teacher does not teach this student"}
	}

	return s.performanceRadarResult(data.StudentID)
}
```

#### 7.5 Регистрация action'ов и HTTP-маршрутов

```go
// internal/app/service.go — диспетчер action'ов
case "student_performance_radar":
	resp := s.studentPerformanceRadar(req.Token)
	resp.ID = req.ID
	return resp
case "teacher_student_performance_radar":
	var data TeacherStudentRadarData
	if err := json.Unmarshal(req.Data, &data); err != nil {
		return Response{ID: req.ID, OK: false, Error: "invalid teacher_student_performance_radar payload"}
	}
	resp := s.teacherStudentPerformanceRadar(req.Token, data)
	resp.ID = req.ID
	return resp
```

```go
// internal/httpserver/server.go — маршруты
fiberApp.Get("/api/student/performance/radar", s.studentPerformanceRadarHandler)
fiberApp.Post("/api/teacher/student/performance/radar", s.teacherStudentPerformanceRadarHandler)

// teacherStudentPerformanceRadarHandler godoc
// @Summary Get a student's performance radar (teacher)
// @Description Teacher gets the per-subject performance radar for a student they teach.
// @Tags grades
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body app.TeacherStudentRadarData true "Student payload"
// @Success 200 {object} app.Response
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Failure 404 {object} app.Response
// @Router /api/teacher/student/performance/radar [post]
func (s *Server) teacherStudentPerformanceRadarHandler(c *fiber.Ctx) error {
	var body app.TeacherStudentRadarData
	return s.gradeActionHandler(c, "http-teacher-student-performance-radar", "teacher_student_performance_radar", &body)
}
```

#### 7.6 Миграция: несколько групп у демо-преподавателя

```sql
-- +goose Up
-- +goose StatementBegin
INSERT INTO lesson_times (lesson_num, start_time, end_time)
VALUES (
    99,
    (CURRENT_TIME + INTERVAL '5 minutes')::time,
    (CURRENT_TIME + INTERVAL '95 minutes')::time
)
ON CONFLICT (lesson_num) DO UPDATE
SET start_time = EXCLUDED.start_time,
    end_time = EXCLUDED.end_time;

DO $$
DECLARE
    v_teacher_id integer;
    v_subject_id integer;
    v_parser_day_idx integer;
    v_inserted integer;
    -- ... вычисление v_parser_day_idx как в прочих demo-schedule миграциях
BEGIN
    SELECT t.teacher_id INTO v_teacher_id
    FROM teachers t JOIN users u ON u.id = t.user_id
    WHERE u.login = 'teacher_test' LIMIT 1;

    SELECT subject_id INTO v_subject_id
    FROM subjects WHERE subject_index = 'TEST-001' LIMIT 1;

    DELETE FROM schedules WHERE teacher_id = v_teacher_id AND lesson_num = 99;

    -- По одной строке расписания на каждую группу, где есть хотя бы один студент
    INSERT INTO schedules (group_id, subject_id, teacher_id, lesson_num, day_idx, week_type, lesson_type)
    SELECT g.group_id, v_subject_id, v_teacher_id, 99, v_parser_day_idx, v_week_type, 'Практика'
    FROM groups g
    WHERE EXISTS (SELECT 1 FROM students s WHERE s.group_id = g.group_id);

    GET DIAGNOSTICS v_inserted = ROW_COUNT;

    -- Фолбэк: если студентов вообще нигде нет — берём первые 5 групп
    IF v_inserted = 0 THEN
        INSERT INTO schedules (group_id, subject_id, teacher_id, lesson_num, day_idx, week_type, lesson_type)
        SELECT g.group_id, v_subject_id, v_teacher_id, 99, v_parser_day_idx, v_week_type, 'Практика'
        FROM groups g
        ORDER BY g.group_id
        LIMIT 5;
    END IF;
END $$;
-- +goose StatementEnd
```
