Feature: Admin Antifraud Monitoring Scenarios

Background:
  * url baseUrl

Scenario: Admin can query fraud logs and top cheaters ranking
  # Login as admin
  Given path 'login'
  And request { login: 'admin_test', password: '123456' }
  When method post
  Then status 200
  * def adminToken = response.result.token

  # Query fraud logs
  Given path 'api/admin/antifraud/logs'
  And header Authorization = 'Bearer ' + adminToken
  And param page = 1
  And param page_size = 10
  When method get
  Then status 200
  And match response.ok == true
  And match response.result.logs == '#array'
  And match response.result.total == '#number'

  # Query top cheaters ranking
  Given path 'api/admin/antifraud/top-cheaters'
  And header Authorization = 'Bearer ' + adminToken
  When method get
  Then status 200
  And match response.ok == true
  And match response.result == '#array'
