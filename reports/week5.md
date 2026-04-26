### Backend (Go + Fiber)

**Файл: http.go**
- Сервер слушает на порту `:8888`
- Использует CORS для разрешения запросов с `localhost:3000`
- Обрабатывает следующие эндпоинты:

```go
POST /register          // Регистрация пользователя
POST /login            // Авторизация
GET  /profile          // Получение профиля (требует токен)
POST /api/teacher/attendance-link    // Создание ссылки на посещаемость (только учитель)
POST /api/student/attendance/confirm // Подтверждение посещаемости (только студент)
```

### Frontend (React + Axios)

**Файл: api.js**
- Использует `axios` для HTTP запросов
- Базовый URL: `http://localhost:8888`

1. **Регистрация/Логин:**
```javascript
// Frontend отправляет:
POST http://localhost:8888/login
{
  "login": "teacher1",
  "password": "pass123",
  "role": "teacher"
}

// Backend возвращает токен в ответе
```

2. **Аутентифицированные запросы:**
```javascript
// Frontend отправляет токен в заголовке:
GET http://localhost:8888/profile
Headers: {
  "Authorization": "your-jwt-token"
}
```