# Недели 16–17 (15–25 июня 2026)

Backend на Go (Fiber + pgx + goose-миграции). Отчёт объединяет прошлую неделю (диаграмма успеваемости + фикс демо-преподавателя) и текущую (почтовая инфраструктура, ролевая модель видимости, новые SQL-запросы под оценки и аналитику). Фронтенд в этом отчёте затронут только там, где это нужно для контекста backend-задачи.

---

## Часть 1. Прошлая неделя: диаграмма успеваемости (radar chart)

Метрика по оси (предмету): набрано баллов / максимум баллов среди работ с прошедшим дедлайном. Оси — все предметы из расписания группы студента, даже без оценок (тогда 0%).

`internal/db/grade_repository.go` — один SQL-запрос берёт предметы из расписания группы и агрегирует баллы только по прошедшим дедлайнам:

```go
func (r *GradeRepository) GetStudentPerformanceRadar(ctx context.Context, studentID int32) ([]SubjectPerformancePoint, error) {
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
	...
}
```

Доступ преподавателя к диаграмме чужого студента ограничен проверкой по расписанию:

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
		 )`,
		teacherID, studentID,
	).Scan(&allowed)
	return allowed, err
}
```

Маршруты: `GET /api/student/performance/radar`, `POST /api/teacher/student/performance/radar`. Плюс миграция `20260614090000_demo_teacher_multi_group.sql`, которая пересобрала демо-расписание `teacher_test`, чтобы оно покрывало все группы со студентами (раньше Android-клиент видел только одну группу).

---

## Часть 2. Эта неделя — backend

### 2.1 Почтовая инфраструктура и сброс пароля

Добавлен `internal/app/mailer.go` — тонкая обёртка над `net/smtp` с собственным `unencryptedAuth` (PLAIN-аутентификация для внутренней SMTP-сети без TLS-проверки имени хоста):

```go
type unencryptedAuth struct {
	identity, username, password string
	host                         string
}

func (a *unencryptedAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	if server.Name != a.host {
		return "", nil, errors.New("wrong host name")
	}
	resp := []byte(a.identity + "\x00" + a.username + "\x00" + a.password)
	return "PLAIN", resp, nil
}
```

Флоу «забыл пароль» в `internal/app/service.go`: токен сброса — 32 случайных байта в hex, живёт 15 минут, письмо уходит асинхронно в горутине, чтобы не блокировать ответ API:

```go
func (s *Service) forgotPassword(data ForgotPasswordData) Response {
	...
	tokenBytes := make([]byte, 32)
	rand.Read(tokenBytes)
	token := hex.EncodeToString(tokenBytes)
	expiresAt := time.Now().Add(15 * time.Minute)

	s.store.Users.CreateResetToken(ctx, dbUser.ID, token, expiresAt)

	email := *dbUser.Email
	go func() {
		if err := s.mailer.SendPasswordReset(email, token); err != nil {
			log.Printf("Failed to send password reset email to %s: %v", email, err)
		}
	}()

	return Response{OK: true, Result: "If the account exists and has a registered email, a reset link has been sent."}
}
```

Намеренно: ответ одинаковый и для существующего, и для несуществующего email — чтобы эндпоинт не давал возможность перебором проверять, какие адреса зарегистрированы в системе.

`resetPassword` проверяет срок токена, при истечении сразу удаляет его (`DeleteResetToken`) и возвращает ту же ошибку `"Invalid or expired token"`, не раскрывая, что токен был просто просрочен, а не подделан. Новая миграция `20260617150000_add_email_and_reset_token.sql` добавила в `users` колонку `email` и таблицу токенов сброса.

### 2.2 Фикс CI: `tsc` падал на вендоренной JS-библиотеке

Не Go, но важный backend-инфраструктурный фикс CI-пайплайна. `npm run check-api` (часть `validate`-джобы в `.github/workflows/main.yml`) гоняет `tsc --project jsconfig.json --noEmit` с `checkJs: true`, и он спотыкался об AMD/CommonJS-детектор модулей в вендоренной `qrcodeGenerator.js`. Поправлено `// @ts-nocheck` первой строкой — без этого ни один PR с этим файлом не мог пройти CI.

### 2.3 Фикс целостности миграций на проде

После очередного `git pull` на сервере `docker compose up` начал валиться на сервисе `migrate`:

```
goose: error: found 1 missing migrations before current version 20260614100000:
	version 20260422180000: migrations/20260422180000_sync_init_schema.sql
```

Файл был закоммичен с версией в имени `20260422180000` (22 апреля), хотя на проде уже стояли применённые миграции `20260614...`/`20260617...` — goose увидел «миграцию из прошлого» после уже накатанных более новых и отказался её применять (защита от несогласованного состояния схемы). Исправлено `git mv` на актуальную версию:

```
migrations/20260422180000_sync_init_schema.sql → migrations/20260619160000_sync_init_schema.sql
```

Проверено локальным прогоном `docker compose up migrate` — `exit 0`, обе миграции (старая и новая по номеру) применились по порядку.

### 2.4 Ролевая модель видимости: зав. кафедрой, декан, админ

Главная backend-задача недели. Понадобилось добавить два новых уровня доступа поверх существующих `student`/`teacher`/`admin`:

| Роль | Видит | Право редактирования |
|---|---|---|
| `teacher` | студентов своих групп (по расписанию) | — |
| `head` (зав. кафедрой) | преподавателей и студентов своей кафедры | нет |
| `dean` (декан) | всё в рамках своего факультета | нет |
| `admin` | всё | да |

**Миграции.** `ALTER TYPE user_role ADD VALUE` нельзя выполнять внутри транзакции — потребовался отдельный файл с директивой `-- +goose NO TRANSACTION`:

```sql
-- +goose NO TRANSACTION
-- +goose Up
ALTER TYPE user_role ADD VALUE IF NOT EXISTS 'head';
ALTER TYPE user_role ADD VALUE IF NOT EXISTS 'dean';
```

Дальше — оргструктура: `faculties` (новая таблица), `lecterns.faculty_id` (внешний ключ), и `org_scopes` — таблица привязки конкретного `head`/`dean` к его охвату:

```sql
CREATE TABLE IF NOT EXISTS org_scopes (
    scope_id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    lectern_id INT REFERENCES lecterns(lectern_id) ON DELETE CASCADE,
    faculty_id INT REFERENCES faculties(faculty_id) ON DELETE CASCADE,
    CONSTRAINT org_scope_target CHECK (lectern_id IS NOT NULL OR faculty_id IS NOT NULL)
);
```

Третья миграция досеяла демо-кафедру и привязала к ней «бесхозные» группы/преподавателей — без этого showcase-данные не давали зав. кафедрой ничего увидеть (охват оказывался пустым).

**Вычисление охвата.** Новый файл `internal/app/supervision.go`. `VisibilityScope` — структура, описывающая, что конкретно разрешено видеть пользователю (флаг «видит всё», список `lectern_id` или список `group_id`), а `scopeForUser` строит её по роли:

```go
type VisibilityScope struct {
	Role       string
	All        bool
	LecternIDs []int32
	GroupIDs   []int32
	Label      string
}

func (s *Service) scopeForUser(ctx context.Context, user User) (VisibilityScope, error) {
	switch user.Role {
	case RoleAdmin:
		return VisibilityScope{Role: RoleAdmin, All: true, Label: "Вся организация"}, nil

	case RoleDean:
		facultyIDs, err := s.scopeFacultyIDs(ctx, user.ID)
		if err != nil {
			return VisibilityScope{}, err
		}
		lecternIDs, err := s.lecternIDsByFaculty(ctx, facultyIDs)
		if err != nil {
			return VisibilityScope{}, err
		}
		label, _ := s.facultyLabel(ctx, facultyIDs)
		return VisibilityScope{Role: RoleDean, LecternIDs: lecternIDs, Label: label}, nil

	case RoleHead:
		lecternIDs, err := s.scopeLecternIDs(ctx, user.ID)
		if err != nil {
			return VisibilityScope{}, err
		}
		label, _ := s.lecternLabel(ctx, lecternIDs)
		return VisibilityScope{Role: RoleHead, LecternIDs: lecternIDs, Label: label}, nil

	case RoleTeacher:
		teacherID, err := s.teacherIDForUser(ctx, user.ID)
		if err != nil {
			return VisibilityScope{}, err
		}
		groupIDs, err := s.teacherGroupIDs(ctx, teacherID)
		if err != nil {
			return VisibilityScope{}, err
		}
		return VisibilityScope{Role: RoleTeacher, GroupIDs: groupIDs, Label: "Мои группы"}, nil

	default:
		return VisibilityScope{}, fmt.Errorf("role %q has no supervisory scope", user.Role)
	}
}
```

Ключевая идея: декан физически не имеет своей строки в `org_scopes` на каждую кафедру — у него привязан только `faculty_id`, а список доступных `lectern_id` разворачивается на лету через `lecternIDsByFaculty`. Это значит, что при добавлении новой кафедры на факультет декан автоматически получает её в охват, без какой-либо ручной синхронизации.

Сами SQL-предикаты строятся динамически под роль вызывающего и параметризуются (никаких склеенных строк с пользовательским вводом — только `= ANY($1)` с массивом):

```go
func scopeStudentPredicate(scope VisibilityScope, alias string) (string, []any) {
	switch {
	case scope.All:
		return "TRUE", nil
	case scope.Role == RoleTeacher:
		return fmt.Sprintf("%s.group_id = ANY($1)", alias), []any{nonNil(scope.GroupIDs)}
	default: // head/dean — студенты, чья группа принадлежит кафедре в охвате
		return fmt.Sprintf("%s.group_id IN (SELECT group_id FROM groups WHERE lectern_id = ANY($1))", alias),
			[]any{nonNil(scope.LecternIDs)}
	}
}
```

Аналогичные `scopeGroupPredicate`/`scopeTeacherPredicate` подставляются в три независимых запроса `staffOverview` (группы, преподаватели, студенты), каждый — с расчётом `% посещаемости` через `attendance_session_students` прямо в SQL (`COUNT(...) FILTER (WHERE status IN ('present','late')) / COUNT(...)`).

Новый маршрут — `GET /api/staff/overview` (action `staff_overview`); студент получает `403`, остальные роли — данные строго в своём охвате. Проверено и через `curl`-сессии под `head_test`/`dean_test`/`student_test`, и через `go build ./...`.

### 2.5 Один запрос вместо N: все оценки студента за раз

Раньше фронтенд просил оценки по одному предмету за вызов (студент вручную вводил `subject_id`). Добавлен `studentAllGrades` (`internal/app/grades.go`) — переиспользует уже существующие репозиторные методы (`GetStudentPerformanceRadar`, `GetStudentGradesBySubject`, `GetSubjectStatsForPrediction`), но агрегирует их в один ответ за один HTTP-запрос:

```go
func (s *Service) studentAllGrades(sessionToken string) Response {
	studentUser, err := s.userBySessionToken(sessionToken)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if studentUser.Role != "student" {
		return Response{OK: false, Error: "forbidden: student role required"}
	}

	studentProfile, err := s.studentProfileByUser(studentUser)
	...
	radar, err := s.store.Grades.GetStudentPerformanceRadar(ctx, studentProfile.ID)
	...

	subjects := make([]map[string]any, 0, len(radar))
	var totalScore, totalMax, totalPassed, gradedWorks, totalWorks int32
	for _, point := range radar {
		grades, _ := s.store.Grades.GetStudentGradesBySubject(ctx, studentProfile.ID, point.SubjectID)
		stats, _ := s.store.Grades.GetSubjectStatsForPrediction(ctx, studentProfile.ID, point.SubjectID)

		for _, g := range grades {
			totalWorks++
			if g.GradedAt != nil {
				gradedWorks++
			}
		}
		totalScore += stats.CurrentScore
		totalMax += stats.TotalMax
		totalPassed += stats.PassedMax

		subjects = append(subjects, map[string]any{
			"subject_id": point.SubjectID, "subject_name": point.SubjectName,
			"percent": point.Percent, "current_score": stats.CurrentScore,
			"total_max": stats.TotalMax, "passed_max": stats.PassedMax,
			"grades": grades,
		})
	}

	return Response{OK: true, Result: map[string]any{
		"student_id": studentProfile.ID,
		"subjects":   subjects,
		"summary": map[string]any{
			"current_score": totalScore, "total_max": totalMax, "passed_max": totalPassed,
			"graded_works": gradedWorks, "total_works": totalWorks,
		},
	}}
}
```

Это N+1-стиль (по запросу на предмет внутри цикла) — осознанный компромисс: число предметов в плане у одного студента редко превышает 10–15, а переиспользование существующих, уже протестированных репозиторных методов было важнее микрооптимизации одним JOIN-запросом. Маршрут — `GET /api/student/grades/all`.

### 2.6 Прочие backend-фиксы недели

- **Реальный фикс бага посещаемости** (не просто диагностика, как было на прошлой итерации): фронтенд никогда не вызывал `api.confirmAttendance` после перехода по ссылке учителя — это чисто frontend-причина, backend-эндпоинт `POST /api/student/attendance/confirm` уже работал корректно и не требовал изменений; добавлен только клиентский вызов.
- Мелкая doc-синхронизация: после каждого нового backend-маршрута (`staff_overview`, `student_all_grades`, password-reset) — Swagger-аннотации в `internal/httpserver/server.go`, чтобы Schemathesis в CI продолжал валидировать контракт.

### 2.7 Проверка (весь объём недели)

- `go build ./...` — чисто после каждого изменения;
- `docker compose up migrate` — все новые миграции применяются по порядку без ошибок (включая `NO TRANSACTION`-миграцию для enum);
- e2e через `curl`: `forgot_password`/`reset_password`, `staff_overview` под `head_test`/`dean_test`/`student_test` (включая проверку `403` у студента), `student_all_grades` под `student_test` — все коды ответов (`200`/`401`/`403`/`400`/`404`) соответствуют ожиданиям;
- `npm run check-api` + `npm run build` — чисто на фронтенд-стороне, синхронизированной с новыми ручками;
- ни одна задача недели не добавила новых Go- или npm-зависимостей.

### 2.8 Итог недели

Закрыт прод-инцидент с целостностью миграций (несогласованная версия в имени файла) и инфраструктурный баг CI (typecheck на вендоренной библиотеке). Добавлена почтовая инфраструктура с безопасным флоу сброса пароля (одинаковый ответ независимо от существования аккаунта, ограниченный TTL токена). Главный результат — полноценная ролевая модель видимости с динамическим вычислением охвата (`scopeForUser`) и параметризованными SQL-предикатами под три новых роли, плюс агрегирующий эндпоинт оценок, устранивший N отдельных запросов с фронтенда.
