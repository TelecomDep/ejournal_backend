Feature: Supervisory Overview & Org Structure Scenarios

Background:
  * url baseUrl

Scenario: Teacher can view supervisory overview scoped to assigned groups
  Given path 'login'
  And request { login: 'teacher_test', password: '123456' }
  When method post
  Then status 200
  * def teacherToken = response.result.token

  Given path 'api/staff/overview'
  And header Authorization = 'Bearer ' + teacherToken
  When method get
  Then status 200
  And match response.ok == true
  And match response.result.role == 'teacher'
  And match response.result.groups == '#array'

Scenario: Staff can search paginated students list
  Given path 'login'
  And request { login: 'teacher_test', password: '123456' }
  When method post
  Then status 200
  * def teacherToken = response.result.token

  Given path 'api/staff/overview/students'
  And header Authorization = 'Bearer ' + teacherToken
  And param page = 1
  And param page_size = 20
  When method get
  Then status 200
  And match response.ok == true
  And match response.result.students == '#array'
  And match response.result.total == '#number'

Scenario: Teacher can get source data for the common rating
  Given path 'login'
  And request { login: 'teacher_test', password: '123456' }
  When method post
  Then status 200
  * def teacherToken = response.result.token

  Given path 'api/staff/ratings/general'
  And header Authorization = 'Bearer ' + teacherToken
  When method get
  Then status 200
  And match response.id == 'http-general-rating'
  And match response.ok == true
  And match response.error == ''
  And match response.result.schema_version == '1.0'
  And match response.result.semester.semester_id == '#number'
  And match response.result.access_control == { role: 'teacher', user_id: '#number', allowed: true }
  And match response.result.departments == '#array'
  And match response.result.subjects == '#array'
  And match response.result.groups == '#array'

Scenario: Admin can view complete organizational structure
  Given path 'login'
  And request { login: 'admin_test', password: '123456' }
  When method post
  Then status 200
  * def adminToken = response.result.token

  Given path 'api/admin/org-structure'
  And header Authorization = 'Bearer ' + adminToken
  When method get
  Then status 200
  And match response.ok == true
  And match response.result == '#array'
