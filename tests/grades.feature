Feature: Grade Management Scenarios

Background:
  * url baseUrl

Scenario: Teacher can create a new grade item for a subject
  Given path 'login'
  And request { login: 'teacher_test', password: '123456' }
  When method post
  Then status 200
  * def teacherToken = response.result.token

  Given path 'api/teacher/subjects'
  And header Authorization = 'Bearer ' + teacherToken
  When method get
  Then status 200
  * def subjectId = response.result.subjects[0].subject_id

  Given path 'api/teacher/grades/items'
  And header Authorization = 'Bearer ' + teacherToken
  And request { subject_id: '#(subjectId)', title: 'Independent Lab 1', max_score: 1, item_type: 'Lab' }
  When method post
  Then status 200
  And match response.ok == true
  And match response.result.item_id == '#number'

Scenario: Teacher can assign a grade to a student
  # --- SETUP: Get student ID & create grade item ---
  Given path 'login'
  And request { login: 'student_test', password: '123456' }
  When method post
  Then status 200
  * def studentToken = response.result.token

  Given path 'profile'
  And header Authorization = 'Bearer ' + studentToken
  When method get
  Then status 200
  * def studentId = response.result.student_id

  Given path 'login'
  And request { login: 'teacher_test', password: '123456' }
  When method post
  Then status 200
  * def teacherToken = response.result.token

  Given path 'api/teacher/subjects'
  And header Authorization = 'Bearer ' + teacherToken
  When method get
  Then status 200
  * def subjectId = response.result.subjects[0].subject_id

  Given path 'api/teacher/grades/items'
  And header Authorization = 'Bearer ' + teacherToken
  And request { subject_id: '#(subjectId)', title: 'Independent Lab 2', max_score: 1, item_type: 'Lab' }
  When method post
  Then status 200
  * def itemId = response.result.item_id

  # --- ACTION: Assign grade ---
  Given path 'api/teacher/grades'
  And header Authorization = 'Bearer ' + teacherToken
  And request { student_id: '#(studentId)', item_id: '#(itemId)', score: 1, comment: 'Good work' }
  When method post
  Then status 200
  And match response.ok == true
  And match response.result.score == 1

Scenario: Student can view their assigned grades
  # --- SETUP: Assign a grade ---
  Given path 'login'
  And request { login: 'student_test', password: '123456' }
  When method post
  Then status 200
  * def studentToken = response.result.token

  Given path 'profile'
  And header Authorization = 'Bearer ' + studentToken
  When method get
  Then status 200
  * def studentId = response.result.student_id

  Given path 'login'
  And request { login: 'teacher_test', password: '123456' }
  When method post
  Then status 200
  * def teacherToken = response.result.token

  Given path 'api/teacher/subjects'
  And header Authorization = 'Bearer ' + teacherToken
  When method get
  Then status 200
  * def subjectId = response.result.subjects[0].subject_id

  Given path 'api/teacher/grades/items'
  And header Authorization = 'Bearer ' + teacherToken
  And request { subject_id: '#(subjectId)', title: 'Independent Lab 3', max_score: 1, item_type: 'Lab' }
  When method post
  Then status 200
  * def itemId = response.result.item_id

  Given path 'api/teacher/grades'
  And header Authorization = 'Bearer ' + teacherToken
  And request { student_id: '#(studentId)', item_id: '#(itemId)', score: 1, comment: 'Excellent' }
  When method post
  Then status 200

  # --- ACTION: Verify grade ---
  Given path 'api/student/grades/all'
  And header Authorization = 'Bearer ' + studentToken
  When method get
  Then status 200
  And match response.ok == true
  * def subjects = response.result.subjects
  * def targetSubject = karate.filter(subjects, function(x){ return x.subject_id == subjectId })[0]
  And match targetSubject != null
  * def grade = karate.filter(targetSubject.grades, function(x){ return x.item_id == itemId })[0]
  And match grade != null
  And match grade.score == 1
  And match grade.max_score == 1
