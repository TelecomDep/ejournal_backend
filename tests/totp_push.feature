Feature: Android Device Push Tokens & TOTP Scenarios

Background:
  * url baseUrl

Scenario: Student can register and list Android device push tokens
  # Login as student
  Given path 'login'
  And request { login: 'student_test', password: '123456' }
  When method post
  Then status 200
  * def studentToken = response.result.token

  # Register FCM device token
  Given path 'api/user/device-token'
  And header Authorization = 'Bearer ' + studentToken
  And request { device_token: 'fcm_test_token_12345', device_name: 'Google Pixel 8', platform: 'android' }
  When method post
  Then status 200
  And match response.ok == true
  And match response.result.status == 'registered'

  # List user device tokens
  Given path 'api/user/device-tokens'
  And header Authorization = 'Bearer ' + studentToken
  When method get
  Then status 200
  And match response.ok == true
  And match response.result == '#array'

  # Teardown: Delete device token
  Given path 'api/user/device-token'
  And header Authorization = 'Bearer ' + studentToken
  And param device_token = 'fcm_test_token_12345'
  When method delete
  Then status 200
  And match response.ok == true
  And match response.result.status == 'deleted'
