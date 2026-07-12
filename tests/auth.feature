Feature: Authentication Scenarios

Background:
  * url baseUrl

Scenario: Unauthenticated request to /profile should fail
  Given path 'profile'
  When method get
  Then status 401
  And match response contains { ok: false, error: 'missing Authorization header' }

Scenario: Login as teacher_test and access profile
  # Login
  Given path 'login'
  And request { login: 'teacher_test', password: '123456' }
  When method post
  Then status 200
  And match response.ok == true
  And match response.result.role == 'teacher'
  * def token = response.result.token

  # Access profile
  Given path 'profile'
  And header Authorization = 'Bearer ' + token
  When method get
  Then status 200
  And match response.ok == true
  And match response.result.role == 'teacher'
  And match response.result.login == 'teacher_test'

Scenario: Login as student_test and access profile
  # Login
  Given path 'login'
  And request { login: 'student_test', password: '123456' }
  When method post
  Then status 200
  And match response.ok == true
  And match response.result.role == 'student'
  * def token = response.result.token

  # Access profile
  Given path 'profile'
  And header Authorization = 'Bearer ' + token
  When method get
  Then status 200
  And match response.ok == true
  And match response.result.role == 'student'
  And match response.result.login == 'student_test'

Scenario: Login with bad credentials should fail
  Given path 'login'
  And request { login: 'student_test', password: 'wrongpassword' }
  When method post
  Then status 401
  And match response.ok == false
  And match response.error == 'wrong password'
