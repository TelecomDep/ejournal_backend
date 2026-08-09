Feature: Notification System Scenarios

Background:
  * url baseUrl

Scenario: User can retrieve notification settings and list notifications
  Given path 'login'
  And request { login: 'student_test', password: '123456' }
  When method post
  Then status 200
  * def studentToken = response.result.token

  Given path 'api/user/notification-settings'
  And header Authorization = 'Bearer ' + studentToken
  When method get
  Then status 200
  And match response.ok == true

  Given path 'api/user/notifications/unread-count'
  And header Authorization = 'Bearer ' + studentToken
  When method get
  Then status 200
  And match response.ok == true
  And match response.result.unread_count == '#number'

Scenario: Admin can create a notification and teardown via HTTP delete handler
  # Login as Admin
  Given path 'login'
  And request { login: 'admin_test', password: '123456' }
  When method post
  Then status 200
  * def adminToken = response.result.token

  # Create Notification
  * def randomSuffix = java.lang.System.currentTimeMillis()
  * def notifTitle = 'System Update ' + randomSuffix

  Given path 'api/admin/notifications'
  And header Authorization = 'Bearer ' + adminToken
  And request { title: '#(notifTitle)', message: 'Scheduled system maintenance.', audience: 'all' }
  When method post
  Then status 201
  And match response.ok == true
  * def notificationId = response.result.notification_id

  # Student views notification list
  Given path 'login'
  And request { login: 'student_test', password: '123456' }
  When method post
  Then status 200
  * def studentToken = response.result.token

  Given path 'api/user/notifications'
  And header Authorization = 'Bearer ' + studentToken
  When method get
  Then status 200
  And match response.ok == true
  And match response.result.items == '#array'

  # TEARDOWN: Admin deletes notification via HTTP handler (Zero Data Pollution)
  Given path 'api/admin/notifications/' + notificationId
  And header Authorization = 'Bearer ' + adminToken
  When method delete
  Then status 200
  And match response.ok == true
