package app

import (
	"strings"
	"time"
)

type StudentScheduleDayData struct {
	Date string `json:"date"`
}

type StudentDayScheduleItem struct {
	LessonNum   int32  `json:"lesson_num"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	SubjectID   int32  `json:"subject_id"`
	SubjectName string `json:"subject_name"`
	TeacherName string `json:"teacher_name"`
	LessonType  string `json:"lesson_type,omitempty"`
	RoomInfo    string `json:"room_info,omitempty"`
	Subgroup    string `json:"subgroup,omitempty"`
}

func startOfWeek(t time.Time) time.Time {
	dayIdx := weekdayToDayIdx(t.Weekday())
	monday := t.AddDate(0, 0, -(int(dayIdx) - 1))
	return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, monday.Location())
}

func (s *Service) studentScheduleForDay(sessionToken string, data StudentScheduleDayData) Response {
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
	if studentProfile.GroupID == nil {
		return Response{OK: false, Error: "student is not assigned to a group"}
	}

	now := time.Now().In(appTimeLocation)

	var target time.Time
	dateStr := strings.TrimSpace(data.Date)
	if dateStr == "" {
		target = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, appTimeLocation)
	} else {
		target, err = time.ParseInLocation("2006-01-02", dateStr, appTimeLocation)
		if err != nil {
			return Response{OK: false, Error: "invalid date format, expected YYYY-MM-DD"}
		}
	}

	weekStart := startOfWeek(now)
	rangeEnd := weekStart.AddDate(0, 0, 14)
	if target.Before(weekStart) || !target.Before(rangeEnd) {
		return Response{OK: false, Error: "date must be within the current or next week"}
	}

	dayIdx := weekdayToDayIdx(target.Weekday())
	weekType := weekTypeByISOParity(target)

	ctx, cancel := s.dbContext()
	defer cancel()

	rows, err := s.store.Pool().Query(
		ctx,
		`SELECT s.lesson_num,
		        lt.start_time,
		        lt.end_time,
		        s.subject_id,
		        sub.name,
		        COALESCE(t.name, ''),
		        COALESCE(s.lesson_type, ''),
		        COALESCE(s.room_info, ''),
		        COALESCE(s.subgroup, '')
		 FROM schedules s
		 JOIN lesson_times lt ON lt.lesson_num = s.lesson_num
		 JOIN subjects sub ON sub.subject_id = s.subject_id
			 LEFT JOIN teachers t ON t.teacher_id = s.teacher_id
			 WHERE s.group_id = $1
			   AND s.day_idx = $2
			   AND s.week_type = $3
			   AND s.semester_id = (
			       SELECT sem.semester_id
			       FROM semesters sem
			       WHERE $4::date BETWEEN
			           (sem.starts_at AT TIME ZONE 'Asia/Novosibirsk')::date
			           AND (sem.ends_at AT TIME ZONE 'Asia/Novosibirsk')::date
			       ORDER BY sem.starts_at DESC
			       LIMIT 1
			   )
			 ORDER BY s.lesson_num`,
		*studentProfile.GroupID,
		dayIdx,
		weekType,
		target.Format("2006-01-02"),
	)
	if err != nil {
		return Response{OK: false, Error: "failed to load schedule"}
	}
	defer rows.Close()

	items := make([]StudentDayScheduleItem, 0)
	for rows.Next() {
		var item StudentDayScheduleItem
		var startClock, endClock time.Time
		if err := rows.Scan(
			&item.LessonNum,
			&startClock,
			&endClock,
			&item.SubjectID,
			&item.SubjectName,
			&item.TeacherName,
			&item.LessonType,
			&item.RoomInfo,
			&item.Subgroup,
		); err != nil {
			return Response{OK: false, Error: "failed to read schedule row"}
		}
		item.StartTime = startClock.Format("15:04")
		item.EndTime = endClock.Format("15:04")
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return Response{OK: false, Error: "failed to iterate schedule rows"}
	}

	return Response{
		OK: true,
		Result: map[string]any{
			"date":      target.Format("2006-01-02"),
			"weekday":   target.Weekday().String(),
			"day_idx":   dayIdx,
			"week_type": weekType,
			"lessons":   items,
		},
	}
}
