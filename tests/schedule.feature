Feature: Student and teacher schedule API

Background:
  * url baseUrl

Scenario: Student can retrieve the day schedule
  Given path 'login'
  And request { login: 'student_test', password: '123456' }
  When method post
  Then status 200
  * def studentToken = response.result.token

  Given path 'api/student/schedule/day'
  And header Authorization = 'Bearer ' + studentToken
  When method get
  Then status 200
  And match response.ok == true
  And match response.result.date == '#string'
  And match response.result.day_idx == '#number'
  And match response.result.week_type == '#number'
  And match response.result.lessons == '#[]'

Scenario: Teacher can retrieve the day schedule with groups
  Given path 'login'
  And request { login: 'teacher_test', password: '123456' }
  When method post
  Then status 200
  * def teacherToken = response.result.token

  Given path 'api/teacher/schedule/day'
  And header Authorization = 'Bearer ' + teacherToken
  When method get
  Then status 200
  And match response.ok == true
  And match response.result.teacher_id == '#number'
  And match response.result.lessons == '#[]'

Scenario: Student cannot retrieve the teacher schedule
  Given path 'login'
  And request { login: 'student_test', password: '123456' }
  When method post
  Then status 200
  * def studentToken = response.result.token

  Given path 'api/teacher/schedule/day'
  And header Authorization = 'Bearer ' + studentToken
  When method get
  Then status 403
  And match response.ok == false
  And match response.error == 'forbidden: teacher role required'

Scenario: Schedule rejects an invalid date
  Given path 'login'
  And request { login: 'teacher_test', password: '123456' }
  When method post
  Then status 200
  * def teacherToken = response.result.token

  Given path 'api/teacher/schedule/day'
  And param date = '20.08.2026'
  And header Authorization = 'Bearer ' + teacherToken
  When method get
  Then status 400
  And match response.ok == false
  And match response.error == 'invalid date format, expected YYYY-MM-DD'
