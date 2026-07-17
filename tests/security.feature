Feature: Security & RBAC Scenarios

Background:
  * url baseUrl

Scenario: Unauthenticated access to protected teacher route returns 401
  Given path 'api/teacher/subjects'
  When method get
  Then status 401
  And match response contains { ok: false, error: 'missing Authorization header' }

Scenario: Unauthenticated access to protected student route returns 401
  Given path 'api/student/grades/all'
  When method get
  Then status 401
  And match response contains { ok: false, error: 'missing Authorization header' }

Scenario: Access with invalid token returns 401
  Given path 'profile'
  And header Authorization = 'Bearer invalid.token.here'
  When method get
  Then status 401
  And match response contains { ok: false }

Scenario: Student accessing teacher route returns 403 Forbidden
  # Login as student
  Given path 'login'
  And request { login: 'student_test', password: '123456' }
  When method post
  Then status 200
  * def studentToken = response.result.token

  # Attempt to access teacher endpoint
  Given path 'api/teacher/subjects'
  And header Authorization = 'Bearer ' + studentToken
  When method get
  Then status 403
  And match response contains { ok: false, error: 'forbidden: teacher role required' }

Scenario: Teacher accessing student route returns 403 Forbidden
  # Login as teacher
  Given path 'login'
  And request { login: 'teacher_test', password: '123456' }
  When method post
  Then status 200
  * def teacherToken = response.result.token

  # Attempt to access student endpoint
  Given path 'api/student/grades/all'
  And header Authorization = 'Bearer ' + teacherToken
  When method get
  Then status 403
  And match response contains { ok: false, error: 'forbidden: student role required' }
