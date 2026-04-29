# EJournal Frontend

Frontend app for `NOC_project_GO` backend.

## Requirements

- Node.js `>=18.18.0` (recommended: `20.x`)
- npm `>=9`

## Run

```bash
nvm use || nvm install
npm install
npm start
```

Default URL: `http://localhost:9001`
Backend URL (env): `REACT_APP_BACKEND_URL=http://localhost:9999`

## Auth Model

- User does not choose role manually.
- Role is derived from `role_hash` during registration and returned by the backend on login.
- Login only needs login and password.
- Test hashes:
  - `TEACHER-HASH-2026`
  - `STUDENT-HASH-2026`

Use `teacher_test / 123456` for quick teacher login.

Attendance test flow:
1. Register/login student with `STUDENT-HASH-2026`.
2. Login teacher (`teacher_test`) and create invite link.
3. Return to student account and confirm attendance with `invite_token`.

## If you previously installed with old Node

```bash
rm -rf node_modules package-lock.json
npm install
npm start
```
