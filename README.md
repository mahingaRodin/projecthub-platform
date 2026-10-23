# ProjectHub

A project management app with JWT login and three roles. It is the application you deploy in the
**DevOps Engineer Practical Exam — Challenge A**.

| Part     | Technology                     | Folder      |
|----------|--------------------------------|-------------|
| Frontend | React 18 + Vite (static files) | `frontend/` |
| Backend  | Go 1.23 + Gin                  | `backend/`  |
| Database | MongoDB 7                      | —           |
| Cache    | Valkey 8+ (optional)           | —           |

> You get the source code only. Dockerfiles, `docker-compose.yml`, NGINX config and Kubernetes manifests are **your** job.

## What the app does

- Projects with a team (owner and members) and a task board (to do / in progress / done).
- **admin:** manages users and every project.
- **manager:** creates projects and manages the ones they own.
- **member:** sees their projects and updates the status of their own tasks.

## Test accounts

The first time the API starts on an empty database, it creates these accounts and a sample project
automatically. Later restarts do not add anything.

| Role    | Email         | Password    |
|---------|---------------|-------------|
| admin   | admin@qt.rw   | Admin@123   |
| manager | manager@qt.rw | Manager@123 |
| member  | member@qt.rw  | Member@123  |

Set `SEED_ENABLED=false` outside the exam so no test accounts are created.

## Configuration

Everything is configured with environment variables. Copy the example files and edit them:
[backend/.env.example](backend/.env.example), [frontend/.env.example](frontend/.env.example).

For local runs the API also reads a `backend/.env` file. Real environment variables, such as Compose
`environment:` or a Kubernetes ConfigMap or Secret, override it. Never commit `.env`.

### Backend

| Variable | Default | Notes |
|---|---|---|
| `JWT_SECRET` | — | **Required**, at least 32 characters (`openssl rand -hex 32`) |
| `MONGO_URI` | `mongodb://localhost:27017` | |
| `MONGO_DB` | `projecthub` | |
| `VALKEY_ADDR` / `VALKEY_PASSWORD` | `localhost:6379` / — | Cache (optional) |
| `PORT` | `8080` | |
| `GIN_MODE` | `debug` | Use `release` |
| `SEED_ENABLED` | `true` | |

### Frontend

| Variable | When | Notes |
|---|---|---|
| `VITE_API_URL` | **build time** | Keep `/api` when NGINX serves the app and the API on the same address |

## Things to know when you deploy

- **Health check:** `GET /api/health` returns `200` when MongoDB is up (`503` otherwise) and reports
  `"cache": "up" | "down" | "disabled"`.
- **Valkey is optional.** If it is down, the app keeps working and reads from MongoDB.
- **The frontend is a single-page app.** Whatever serves `dist/` must return `index.html` for unknown paths
  such as `/projects/123`.
- `VITE_API_URL` is fixed when you run `npm run build`. Changing it later needs a rebuild.
- The API is a single static binary: `CGO_ENABLED=0 go build`.

## Build and run locally

Requirements: Go 1.23+, Node.js 20+, MongoDB 7, Valkey (optional).

```bash
# backend → http://localhost:8080
cd backend
cp .env.example .env          # set JWT_SECRET and MONGO_URI
go run .                      # or: CGO_ENABLED=0 go build -o projecthub-api .

# frontend → http://localhost:5173
cd frontend
npm install
npm run dev                   # /api is forwarded to :8080
npm run build                 # static files in dist/
```

## API

All paths start with `/api`. Send `Authorization: Bearer <token>` except for health, register and login.

| Method | Path | Who | Purpose |
|---|---|---|---|
| POST | `/auth/register` · `/auth/login` | everyone | Get a token |
| GET | `/auth/me` · `/stats` | logged in | Current user, dashboard |
| GET · POST | `/projects` | all · admin, manager | List / create projects |
| GET · PUT · DELETE | `/projects/:id` | team · owner, admin | View / edit (incl. members and `ownerId`) / delete |
| GET · POST | `/projects/:id/tasks` | team · owner, admin | List / create tasks |
| PATCH · DELETE | `/tasks/:id` | owner, admin (assignee: status only) | Update / delete a task |
| GET · PATCH · DELETE | `/users`, `/users/:id/role`, `/users/:id` | admin (list: also manager) | Manage users |
| GET | `/health` | everyone | Health check |

## Project layout

```
backend/   Go API: internal/ controller/ service/ repository/ models/ cache/ middleware/ router/
frontend/  React app: src/pages/, src/components/, src/api.js
```
