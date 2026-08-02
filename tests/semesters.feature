Feature: Semester Management & Query API Scenarios

Background:
  * url baseUrl

Scenario: Anyone can list available semesters
  Given path 'api/semesters'
  When method get
  Then status 200
  And match response.ok == true
  And match response.result.items == '#[]'
  * def firstSemester = response.result.items[0]
  And match firstSemester.semester_id == '#number'
  And match firstSemester.academic_year == '#string'
  And match firstSemester.term_num == '#number'
  And match firstSemester.status == '#string'

Scenario: Retrieve current active semester
  Given path 'api/semesters/current'
  When method get
  Then status 200
  And match response.ok == true
  And match response.result.semester == '#object'
  * def currentSem = response.result.semester
  And match currentSem.semester_id == '#number'
  And match currentSem.is_current == true
  And match currentSem.status == 'open'

Scenario: Non-admin users cannot create semesters (RBAC check)
  Given path 'login'
  And request { login: 'teacher_test', password: '123456' }
  When method post
  Then status 200
  * def teacherToken = response.result.token

  Given path 'api/admin/semesters'
  And header Authorization = 'Bearer ' + teacherToken
  And request { academic_year: '2028/2029', term_num: 1, name: '2028/2029, Fall', starts_at: '2028-09-01T00:00:00Z', ends_at: '2029-01-31T23:59:59Z' }
  When method post
  Then status 403
  And match response.ok == false
  And match response.error == 'forbidden: admin role required'

Scenario: Admin semester creation validation & lifecycle transition
  # --- SETUP: Login as admin ---
  Given path 'login'
  And request { login: 'admin_test', password: '123456' }
  When method post
  Then status 200
  * def adminToken = response.result.token

  # --- VALIDATION: Invalid academic_year format ---
  Given path 'api/admin/semesters'
  And header Authorization = 'Bearer ' + adminToken
  And request { academic_year: '2028-2029', term_num: 1, name: 'Invalid Year', starts_at: '2028-09-01T00:00:00Z', ends_at: '2029-01-31T23:59:59Z' }
  When method post
  Then status 400
  And match response.ok == false
  And match response.error == 'academic_year must have format YYYY/YYYY'

  # --- CREATE: Create planned future semester ---
  Given path 'api/admin/semesters'
  And header Authorization = 'Bearer ' + adminToken
  And request { academic_year: '2028/2029', term_num: 1, name: '2028/2029, осенний семестр', starts_at: '2028-09-01T00:00:00Z', ends_at: '2029-01-31T23:59:59Z', status: 'planned' }
  When method post
  Then status 200
  And match response.ok == true
  * def newSemesterId = response.result.semester.semester_id
  And match response.result.semester.status == 'planned'

  # --- ACTION: Attempt to activate future semester (should fail validation) ---
  Given path 'api/admin/semesters/' + newSemesterId + '/activate'
  And header Authorization = 'Bearer ' + adminToken
  And request { semester_id: '#(newSemesterId)' }
  When method patch
  Then status 409
  And match response.ok == false
  And match response.error == 'semester has not started'

  # --- ACTION: Archive closed semester (semester 1) ---
  Given path 'api/admin/semesters/1/archive'
  And header Authorization = 'Bearer ' + adminToken
  And request { semester_id: 1 }
  When method patch
  Then status 200
  And match response.ok == true
  And match response.result.semester.status == 'archived'

  # --- TEARDOWN: Delete created planned semester to prevent server data pollution ---
  Given path 'api/admin/semesters/' + newSemesterId
  And header Authorization = 'Bearer ' + adminToken
  When method delete
  Then status 200
  And match response.ok == true
  And match response.result.deleted == true
