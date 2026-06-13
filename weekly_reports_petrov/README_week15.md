# Неделя 15 (8-13 июня 2026)

## Клиентский UX преподавателя и студента: каскадные выборы, тепловая карта посещаемости, подтверждения действий и сводка по группе

### 1. Цель недели

На этой неделе фокус сместился с самого API на удобство интерфейса для преподавателя и студента. Раньше во многих формах преподаватель вручную вводил числовые `ID` групп и студентов, тепловая карта посещаемости студента не была связана с БД, а итоговая ведомость содержала непонятные подписи.

Ключевые задачи:
- убрать ручной ввод `ID` групп/студентов у преподавателя — заменить каскадными выпадающими списками "Предмет → Группа → Студент";
- подключить тепловую карту посещаемости студента к реальным данным БД и починить смещение дат из-за UTC;
- показывать понятные названия групп вместо числовых `ID` во вкладке «Посещаемость»;
- добавить подтверждение (`confirm`) перед каждым действием преподавателя, которое меняет данные;
- сделать понятными подписи в «Ведомости студента» (было неясно, что значит «Прошло»);
- добавить новую вкладку «Сводка по группе» — посещаемость и успеваемость каждого студента в одной таблице;
- починить сборку `docker compose` после внесённых изменений.

### 2. Каскадный выбор "Предмет → Группа → Студент"

Во вкладке «Оценки» вместо текстовых полей `ID студента` и `ID предмета` теперь три связанных `<select>`. При выборе предмета подтягиваются его группы (с названиями, не `ID`), при выборе группы — состав студентов через уже существующую ручку `/api/teacher/attendance/group`.

Добавлен общий helper `groupsOf(subject)`, который приводит группы предмета к виду `[{ id, name }]` (бэкенд теперь отдаёт `groups`, поле `group_ids` оставлено как фолбэк для совместимости).

### 3. Чекбоксы групп вместо текстового поля во вкладке «Посещаемость»

Поле «ID групп через запятую» при создании сессии посещаемости заменено на список чипов-чекбоксов с названиями групп (`group-checklist` / `group-chip`), полученных через тот же `groupsOf`.

### 4. Тепловая карта посещаемости студента

На дашборде студента `AttendanceGrid` + пустые `DataTable` заменены на `AttendanceHeatmap`, подключённую к `attendanceHeatmapData` из БД (`/api/student/attendance/history`). Дополнительно исправлено смещение дат на один день: `date.toISOString()` переводит дату в UTC и может "съезжать" на соседний день, поэтому добавлен локальный форматтер `toLocalDateStr`.

### 5. Подтверждения действий преподавателя

Перед каждым изменяющим действием теперь показывается `window.confirm()` с человекочитаемым текстом (название работы/студента/групп, а не `ID`):
- создание сессии посещаемости — со списком названий групп и временем действия ссылки;
- создание контрольной точки — с названием работы, баллами и предметом;
- выставление оценки — с ФИО студента, баллом и названием работы.

После успешного действия выводится конкретное сообщение (например, «Оценка сохранена: Иванов И.И. — 8 б. за «Лабораторная работа 1»»), а не общее «Сохранено».

### 6. Понятная «Ведомость студента»

Карточки сводки переименованы и снабжены пояснениями (`title`-tooltip + текст под таблицей):
- «Набрано» → **Набрано баллов** — сколько студент уже набрал;
- «План» → **Максимум по предмету** — максимум баллов по всем работам семестра;
- «Прошло» → **Ожидалось к сроку** — сколько баллов уже можно было набрать по работам с прошедшим дедлайном. Если «Набрано» заметно меньше — у студента отставание.

### 7. Новая ручка и вкладка «Сводка по группе»

Добавлен новый endpoint `POST /api/teacher/group/performance` (action `teacher_group_performance`), который одним запросом отдаёт по каждому студенту группы:
- посещаемость (посещено/всего занятий, процент);
- успеваемость (набрано/максимум баллов, процент);
- средние показатели по группе.

На фронтенде вкладка «Посещаемость группы» переделана в «Сводка по группе»: каскад "Предмет → Группа", карточки со средними значениями и таблица с цветными бейджами (`badge-ok` ≥75%, `badge-warn` ≥50%, `badge-bad` <50%).

Также добавлена сопутствующая ручка `POST /api/teacher/attendance/student/history` для детальной истории посещений конкретного студента по предмету.

### 8. Фикс сборки docker compose

После правок фронтенда `docker compose build web` падал на этапе `npm run build` с ошибкой:
```
Line 214:5:  Definition for rule 'react-hooks/exhaustive-deps' was not found  react-hooks/exhaustive-deps
```
Причина: в коде остались комментарии `// eslint-disable-next-line react-hooks/exhaustive-deps`, а сам плагин `eslint-plugin-react-hooks` в проекте не подключён — CRA считает такие ESLint-ошибки фатальными для билда. Комментарии удалены, после чего сборка прошла успешно.

### 9. Тестирование и валидация

- `go build ./...`, `go vet ./...` — без ошибок;
- `docker compose build ejournal-backend web` — оба сервиса собраны успешно ("Compiled successfully.");
- `docker compose up -d` — все контейнеры в статусе `Up` (`postgres` healthy);
- ручная проверка через `curl`:
  - `POST /login` (`teacher_test` / `123456`) → токен получен;
  - `POST /api/teacher/group/performance {"group_id":2,"subject_id":1}` → корректный ответ с посещаемостью/успеваемостью по каждому студенту и средними по группе;
- фронтенд `http://localhost:9999/` отвечает `200`.

### 10. Итог недели

Интерфейс преподавателя стал значительно дружелюбнее: ручной ввод `ID` практически исчез из основных сценариев, действия подтверждаются и сопровождаются понятными сообщениями, а в одной таблице теперь видна и посещаемость, и успеваемость группы. У студента тепловая карта посещаемости отображает реальные данные из БД без смещения дат.

### 11. Ключевые вставки кода

#### 11.1 Helper для групп предмета (frontend)

```jsx
// Возвращает список групп предмета в виде [{ id, name }].
// Бэкенд отдаёт поле groups; group_ids оставлен как фолбэк для совместимости.
const groupsOf = (subject) => {
  if (!subject) {
    return [];
  }
  if (Array.isArray(subject.groups) && subject.groups.length > 0) {
    return subject.groups
      .map((group) => ({ id: Number(group.id), name: group.name || `Группа ${group.id}` }))
      .filter((group) => Number.isFinite(group.id) && group.id > 0);
  }
  return (Array.isArray(subject.group_ids) ? subject.group_ids : [])
    .map((id) => ({ id: Number(id), name: `Группа ${id}` }))
    .filter((group) => Number.isFinite(group.id) && group.id > 0);
};
```



#### 11.2 SQL-запрос сводки по группе (backend)

```go
// GetGroupSubjectPerformance returns, for each student of the group, combined
// attendance (sessions) and grade totals on the given subject.
func (r *AttendanceRepository) GetGroupSubjectPerformance(
	ctx context.Context,
	teacherID int32,
	groupID int32,
	subjectID int32,
) ([]GroupSubjectPerformanceRow, error) {
	rows, err := r.pool.Query(
		ctx,
		`WITH scoped_sessions AS (
		     SELECT s.session_id
		     FROM attendance_sessions s
		     INNER JOIN attendance_session_groups sg ON sg.session_id = s.session_id
		     WHERE s.teacher_id = $1 AND sg.group_id = $2 AND s.subject_id = $3
		   ),
		   att AS (
		     SELECT ass.student_id,
		            COUNT(*)::INTEGER AS total_sessions,
		            SUM(CASE WHEN ass.status = 'present' THEN 1 ELSE 0 END)::INTEGER AS attended_sessions
		     FROM attendance_session_students ass
		     INNER JOIN scoped_sessions ss ON ss.session_id = ass.session_id
		     WHERE ass.group_id_snapshot = $2
		     GROUP BY ass.student_id
		   ),
		   grd AS (
		     SELECT st.student_id,
		            COALESCE(SUM(gi.max_score), 0)::INTEGER AS total_max,
		            COALESCE(SUM(CASE WHEN gi.deadline < now() THEN gi.max_score ELSE 0 END), 0)::INTEGER AS passed_max,
		            COALESCE(SUM(g.score), 0)::INTEGER AS current_score
		     FROM students st
		     LEFT JOIN grade_items gi ON gi.subject_id = $3
		     LEFT JOIN grades g ON g.item_id = gi.item_id AND g.student_id = st.student_id
		     WHERE st.group_id = $2
		     GROUP BY st.student_id
		   )
		   SELECT st.student_id, st.student_name,
		          COALESCE(att.total_sessions, 0), COALESCE(att.attended_sessions, 0),
		          COALESCE(grd.total_max, 0), COALESCE(grd.passed_max, 0), COALESCE(grd.current_score, 0)
		   FROM students st
		   LEFT JOIN att ON att.student_id = st.student_id
		   LEFT JOIN grd ON grd.student_id = st.student_id
		   WHERE st.group_id = $2
		   ORDER BY st.student_name, st.student_id`,
		teacherID, groupID, subjectID,
	)
	// ... сканирование строк в []GroupSubjectPerformanceRow
}
```

#### 11.3 Сервисный метод и регистрация ручки

```go
// groupPerformanceForTeacher returns a combined overview for a group on a
// subject: per-student attendance and grade totals plus group averages.
func (s *Service) groupPerformanceForTeacher(sessionToken string, data GroupPerformanceData) Response {
	// ... проверка роли teacher, ensureTeacherSubjectAccess, загрузка group/subject

	rows, err := s.store.Attendance.GetGroupSubjectPerformance(ctx, teacherProfile.ID, data.GroupID, data.SubjectID)

	students := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		attendancePercent := 0.0
		if row.TotalSessions > 0 {
			attendancePercent = float64(row.AttendedSessions) * 100 / float64(row.TotalSessions)
		}
		gradePercent := 0.0
		if row.TotalMax > 0 {
			gradePercent = float64(row.CurrentScore) * 100 / float64(row.TotalMax)
		}
		students = append(students, map[string]any{
			"student_id": row.StudentID, "student_name": row.StudentName,
			"total_sessions": row.TotalSessions, "attended_sessions": row.AttendedSessions,
			"attendance_percent": attendancePercent,
			"current_score": row.CurrentScore, "total_max": row.TotalMax, "passed_max": row.PassedMax,
			"grade_percent": gradePercent,
		})
	}

	return Response{OK: true, Result: map[string]any{
		"group_id": data.GroupID, "group_name": group.GroupName,
		"subject_id": subject.ID, "subject_name": subject.Name,
		"timezone": "Asia/Novosibirsk",
		"students": students,
		"summary": map[string]any{
			"students_count": len(students), "sessions_count": sessionsCount,
			"avg_attendance_percent": avgAttendance, "avg_grade_percent": avgGrade,
		},
	}}
}
```

```go
// Регистрация маршрута
fiberApp.Post("/api/teacher/group/performance", s.teacherGroupPerformanceHandler)

// Диспетчер action'ов
case "teacher_group_performance":
	var data GroupPerformanceData
	if err := json.Unmarshal(req.Data, &data); err != nil {
		return Response{ID: req.ID, OK: false, Error: "invalid teacher_group_performance payload"}
	}
	resp := s.groupPerformanceForTeacher(req.Token, data)
	resp.ID = req.ID
	return resp
```

#### 11.4 Таблица сводки с цветными бейджами (frontend)

```jsx
<tr key={student.student_id}>
  <td>{student.student_name}</td>
  <td>
    {student.attended_sessions}/{student.total_sessions}
    {' · '}
    <span className={`badge ${attendance >= 75 ? 'badge-ok' : attendance >= 50 ? 'badge-warn' : 'badge-bad'}`}>
      {Number.isFinite(attendance) ? `${attendance.toFixed(0)}%` : '—'}
    </span>
  </td>
  <td>
    <span className={`badge ${grade >= 75 ? 'badge-ok' : grade >= 50 ? 'badge-warn' : 'badge-bad'}`}>
      {Number.isFinite(grade) ? `${grade.toFixed(0)}%` : '—'}
    </span>
  </td>
  <td>{student.current_score} / {student.total_max}</td>
</tr>
```
