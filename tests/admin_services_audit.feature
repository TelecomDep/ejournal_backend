Feature: Admin Launchpad, Audit Logs, and Maintenance Mode Scenarios

Background:
  * url baseUrl

Scenario: Admin can retrieve external service launchpad links
  Given path 'login'
  And request { login: 'admin_test', password: '123456' }
  When method post
  Then status 200
  * def adminToken = response.result.token

  Given path 'api/admin/services'
  And header Authorization = 'Bearer ' + adminToken
  When method get
  Then status 200
  And match response.ok == true
  And match response.result == '#array'
  And match response.result[0].id == '#string'
  And match response.result[0].external_url == '#string'

Scenario: Admin can query audit logs and toggle system maintenance mode
  Given path 'login'
  And request { login: 'admin_test', password: '123456' }
  When method post
  Then status 200
  * def adminToken = response.result.token

  # Query audit logs
  Given path 'api/admin/audit-logs'
  And header Authorization = 'Bearer ' + adminToken
  And param page = 1
  And param page_size = 10
  When method get
  Then status 200
  And match response.ok == true
  And match response.result.logs == '#array'

  # Get maintenance status
  Given path 'api/admin/system/maintenance'
  And header Authorization = 'Bearer ' + adminToken
  When method get
  Then status 200
  And match response.ok == true
  And match response.result.enabled == '#boolean'

  # Toggle maintenance status
  Given path 'api/admin/system/maintenance'
  And header Authorization = 'Bearer ' + adminToken
  And request { enabled: false, message: 'System operating normally' }
  When method post
  Then status 200
  And match response.ok == true
  And match response.result.enabled == false
