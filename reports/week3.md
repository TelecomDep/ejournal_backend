# week3

## Запрос данных с сервера (фронтенд)

### 1. **WebSocket соединение (useWebSocket.js)**
Хук управляет WebSocket соединением, отправляет запросы и принимает ответы:

```javascript
// Отправка запроса
const sendMessage = useCallback((message) => {
  if (ws.current && ws.current.readyState === WebSocket.OPEN) {
    // Генерируем уникальный ID для запроса
    const request = {
      ...message,
      id: message.id || `${message.action}_${Date.now()}`
    };
    ws.current.send(JSON.stringify(request));
  } else {
    console.error('WebSocket is not connected');
  }
}, []);

// Получение ответа
ws.current.onmessage = (event) => {
  setLastMessage(event.data);
};
```

### 2. **Инициализация запросов (App.jsx)**
При подключении WebSocket отправляются запросы на получение данных:

```javascript
useEffect(() => {
  // Запрос данных пользователя после логина
  if (connectionStatus === 'connected') {
    sendMessage({
      action: 'getUserData',
      data: {}
    });
    
    sendMessage({
      action: 'getStudentsList',
      data: {}
    });
    
    sendMessage({
      action: 'getAttendance',
      data: {}
    });
  }
}, [connectionStatus, sendMessage]);
```

### 3. **Обработка ответов (App.jsx)**
Полученные сообщения парсятся и обрабатываются по ID запроса:

```javascript
useEffect(() => {
  if (lastMessage) {
    const response = JSON.parse(lastMessage);
    if (response.ok) {
      switch (response.id) {
        case 'getUserData':
          setUserData(response.result);
          break;
        case 'getStudentsList':
          setStudentsData(response.result);
          break;
        case 'getAttendance':
          setAttendanceData(response.result);
          break;
        default:
          break;
      }
    } else {
      console.error('Error:', response.error);
    }
  }
}, [lastMessage]);
```

## Формат взаимодействия

**Исходящий запрос:**
```json
{
  "id": "getUserData_1234567890",
  "action": "getUserData",
  "token": "user_token_here", 
  "data": {}
}
```

**Входящий ответ:**
```json
{
  "id": "getUserData_1234567890",
  "ok": true,
  "result": {
    // данные пользователя
  },
  "error": ""
}
```

**В случае ошибки:**
```json
{
  "id": "getUserData_1234567890",
  "ok": false,
  "error": "Описание ошибки"
}
```