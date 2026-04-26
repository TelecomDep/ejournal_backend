#### 1. Архитектура коммуникации

Взаимодействие построено по классической клиент-серверной архитектуре с использованием **REST API** через библиотеку **Axios**.

- **Клиент:** React-приложение (файл `App.jsx`).
- **Сервер:** Локальный или удаленный бэкенд (URL настраивается через переменную окружения `REACT_APP_BACKEND_URL`, по умолчанию `http://localhost:8888`).
- **Формат данных:** JSON.
- **Способ аутентификации:** **Token-Based Authentication**. Токен передается в заголовке `Authorization` после успешного входа.

#### 2. Уровень абстракции запросов (`api.js`)

Файл `api.js` служит **сервисным слоем (service layer)**. Он скрывает детали настройки HTTP (URL, хедеры) от компонентов React.

**Как происходит соединение:**
1.  **Настройка Axios:** Используется `axios.post` или `axios.get`.
2.  **Базовый URL:** Все запросы идут на адрес, определенный в `BACKEND_URL`.
3.  **Передача токена:** В защищенных методах (`getProfile`, `createAttendanceLink`) токен вставляется в хедер `'Authorization': token`.

**Методы, реализованные в `api.js`:**
*   `login(login, password, role)`: `POST /login` (получение токена).
*   `getProfile(token)`: `GET /profile` (получение данных юзера по токену).
*   `register(...)`: `POST /register` (регистрация, пока не вызывается из `App.jsx` напрямую, но готова).
*   `createAttendanceLink(...)`: `POST /api/teacher/attendance-link` (для учителя).
*   `confirmAttendance(...)`: `POST /api/student/attendance/confirm` (для студента).

#### 3. Жизненный цикл соединения в `App.jsx`

Вся логика управления соединением сосредоточена в корневом компоненте `App`.

**Этап 1: Вход (Login)**
*   **Действие пользователя:** Ввод логина/пароля в `LoginPage`.
*   **Вызов:** `handleLogin` -> `api.login()`.
*   **HTTP-запрос:**
    ```http
    POST http://localhost:8080/login
    Content-Type: application/json

    {
        "login": "user",
        "password": "pass",
        "role": "teacher"
    }
    ```
*   **Ответ сервера:** `{ "token": "some_secret_token", "login": "user", "role": "teacher" }`.
*   **Сохранение состояния:** Токен сохраняется в `localStorage` (ключ `'token'`) и в стейт React (`setToken`). Это позволяет пережить перезагрузку страницы.

**Этап 2: Восстановление сессии (useEffect)**
*   При загрузке страницы `App` проверяет `localStorage.getItem('token')`.
*   **Если токен найден:**
    1.  Вызывается `api.getProfile(token)`.
    2.  **HTTP-запрос:**
        ```http
        GET http://localhost:8888/profile
        Authorization: some_secret_token
        ```



#### 4. Итоговая схема потока данных

1.  **Пользователь** -> Вводит данные.
2.  **React (api.js)** -> **POST /login** -> **Бэкенд**.
3.  **Бэкенд** -> Возвращает **Token**.
4.  **React (App.jsx)** -> Сохраняет Token в **localStorage**.
5.  **React (App.jsx)** -> **GET /profile** (с Token в хедере) -> **Бэкенд**.
6.  **Бэкенд** -> Возвращает данные пользователя.
7.  **React** -> Рендерит `ProfileSquare`, `PersonalAccount` и **Статические таблицы** (не из БД).
