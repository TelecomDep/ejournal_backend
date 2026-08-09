Feature: Admin User & System Management Scenarios

Background:
  * url baseUrl

Scenario: Admin can retrieve system stats and role permission matrix
  # Login as admin
  Given path 'login'
  And request { login: 'admin_test', password: '123456' }
  When method post
  Then status 200
  * def adminToken = response.result.token

  # Retrieve system stats
  Given path 'api/admin/stats'
  And header Authorization = 'Bearer ' + adminToken
  When method get
  Then status 200
  And match response.ok == true
  And match response.result.active_semester == '#string'
  And match response.result.total_groups == '#number'

  # Retrieve role permission matrix
  Given path 'api/admin/roles'
  And header Authorization = 'Bearer ' + adminToken
  When method get
  Then status 200
  And match response.ok == true
  And match response.result == '#array'

Scenario: Admin can list system users with pagination
  Given path 'login'
  And request { login: 'admin_test', password: '123456' }
  When method post
  Then status 200
  * def adminToken = response.result.token

  Given path 'api/admin/users'
  And header Authorization = 'Bearer ' + adminToken
  And param page = 1
  And param page_size = 10
  When method get
  Then status 200
  And match response.ok == true
  And match response.result.items == '#array'
  And match response.result.pagination.total == '#number'

Scenario: Admin can create a student user and archive them via HTTP handler (Zero Data Pollution)
  Given path 'login'
  And request { login: 'admin_test', password: '123456' }
  When method post
  Then status 200
  * def adminToken = response.result.token

  # Generate random login suffix
  * def randomSuffix = java.lang.System.currentTimeMillis()
  * def testLogin = 'karate_user_' + randomSuffix

  # Create user
  Given path 'api/admin/users'
  And header Authorization = 'Bearer ' + adminToken
  And request { login: '#(testLogin)', password: 'TestPassword123', role: 'student', full_name: 'Karate Test Student' }
  When method post
  Then status 201
  And match response.ok == true
  * def createdUserId = response.result.user_id

  # Fetch single user
  Given path 'api/admin/users/' + createdUserId
  And header Authorization = 'Bearer ' + adminToken
  When method get
  Then status 200
  And match response.ok == true
  And match response.result.login == testLogin

  # TEARDOWN: Archive user via HTTP handler (Zero Data Pollution)
  Given path 'api/admin/users/' + createdUserId
  And header Authorization = 'Bearer ' + adminToken
  When method delete
  Then status 200
  And match response.ok == true
  And match response.result.status == 'archived'
