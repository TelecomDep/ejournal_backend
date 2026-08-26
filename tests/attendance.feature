Feature: Attendance Management Scenarios

Background:
  * url baseUrl

Scenario: Teacher can retrieve their assigned subjects
  Given path 'login'
  And request { login: 'teacher_test', password: '123456' }
  When method post
  Then status 200
  * def teacherToken = response.result.token

  Given path 'api/teacher/subjects'
  And header Authorization = 'Bearer ' + teacherToken
  When method get
  Then status 200
  And match response.ok == true
  And match response.result.subjects != []
  And match response.result.subjects[0].subject_id == '#number'

Scenario: Teacher can create an attendance link
  Given path 'login'
  And request { login: 'teacher_test', password: '123456' }
  When method post
  Then status 200
  * def teacherToken = response.result.token

  Given path 'api/teacher/subjects'
  And header Authorization = 'Bearer ' + teacherToken
  When method get
  Then status 200
  * def subject = response.result.subjects[0]
  * def subjectId = subject.subject_id
  * def groupIds = subject.group_ids

  Given path 'api/teacher/attendance-link'
  And header Authorization = 'Bearer ' + teacherToken
  And request { subject_id: '#(subjectId)', group_ids: '#(groupIds)', expires_minutes: 15 }
  When method post
  Then status 200
  And match response.ok == true
  And match response.result.invite_token == '#string'
  And match response.result.lesson_id == '#string'

Scenario: Student can confirm attendance with a valid invite token
  # --- SETUP: Teacher creates the session ---
  Given path 'login'
  And request { login: 'teacher_test', password: '123456' }
  When method post
  Then status 200
  * def teacherToken = response.result.token

  Given path 'api/teacher/subjects'
  And header Authorization = 'Bearer ' + teacherToken
  When method get
  Then status 200
  * def subject = response.result.subjects[0]
  * def subjectId = subject.subject_id
  * def groupIds = subject.group_ids

  Given path 'api/teacher/attendance-link'
  And header Authorization = 'Bearer ' + teacherToken
  And request { subject_id: '#(subjectId)', group_ids: '#(groupIds)', expires_minutes: 15 }
  When method post
  Then status 200
  * def inviteToken = response.result.invite_token

  # --- ACTION: Student confirms attendance ---
  Given path 'login'
  And request { login: 'student_test', password: '123456' }
  When method post
  Then status 200
  * def studentToken = response.result.token

  Given path 'api/student/attendance/confirm'
  And header Authorization = 'Bearer ' + studentToken
  And request { invite_token: '#(inviteToken)' }
  When method post
  Then status 200
  And match response.ok == true
  And match response.result.attendance == 'confirmed'
  And match response.result.subject_id == subjectId

Scenario: Teacher can view and correct the live attendance roster
  Given path 'login'
  And request { login: 'teacher_test', password: '123456' }
  When method post
  Then status 200
  * def teacherToken = response.result.token

  Given path 'api/teacher/subjects'
  And header Authorization = 'Bearer ' + teacherToken
  When method get
  Then status 200
  * def subject = response.result.subjects[0]
  * def subjectId = subject.subject_id
  * def groupIds = subject.group_ids

  Given path 'api/teacher/attendance-link'
  And header Authorization = 'Bearer ' + teacherToken
  And request { subject_id: '#(subjectId)', group_ids: '#(groupIds)', expires_minutes: 15 }
  When method post
  Then status 200
  * def lessonId = response.result.lesson_id

  Given path 'api/teacher/attendance/session/roster'
  And param lesson_id = lessonId
  And header Authorization = 'Bearer ' + teacherToken
  When method get
  Then status 200
  And match response.ok == true
  And match response.result.lesson_id == parseInt(lessonId)
  And match response.result.students != []
  And match each response.result.students contains { student_id: '#number', student_name: '#string', group_id: '#number', group_name: '#string', status: '#string' }
  * def studentId = response.result.students[0].student_id
  * def lessonIdInt = parseInt(lessonId)
  * def studentIdInt = parseInt(studentId)

  Given path 'api/teacher/attendance/mark'
  And header Authorization = 'Bearer ' + teacherToken
  And request { lesson_id: '#(lessonIdInt)', student_id: '#(studentIdInt)', status: 'late' }
  When method post
  Then status 200
  And match response.ok == true
  And match response.result.status == 'late'

  Given path 'api/teacher/attendance/session/roster'
  And param lesson_id = lessonId
  And header Authorization = 'Bearer ' + teacherToken
  When method get
  Then status 200
  And match response.result.students contains deep { student_id: '#(studentIdInt)', status: 'late', marked_by: 'teacher' }

Scenario: Student can view attendance percentages for the current semester
  Given path 'login'
  And request { login: 'student_test', password: '123456' }
  When method post
  Then status 200
  * def studentToken = response.result.token

  Given path 'api/student/attendance/summary'
  And header Authorization = 'Bearer ' + studentToken
  When method get
  Then status 200
  And match response.ok == true
  And match response.result.semester_id == '#number'
  And match response.result.summary.attendance_percent == '#number'
  And match response.result.summary.total_sessions == '#number'
  And match response.result.subjects == '#[]'
  And match each response.result.subjects contains { subject_id: '#number', subject_name: '#string', attendance_percent: '#number', total_sessions: '#number', attended_sessions: '#number', excused_sessions: '#number', missed_sessions: '#number' }
