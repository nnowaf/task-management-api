# GDCPAY — Task Management API (Multi-User)

A multi-user task management REST API built with **Go + Fiber**, **raw SQL on PostgreSQL (pgx)**, JWT auth,
**idempotent** task creation, **transactional** task assignment, and **structured JSON logging**.
---

## Table of contents

- [Tech stack](#tech-stack)
- [Quick start (Docker)](#quick-start-docker)
- [Run locally](#run-locally)
- [Environment variables](#environment-variables)
- [Project structure](#project-structure)
- [API endpoints](#api-endpoints)
- [Authentication](#authentication)
- [Idempotency (`POST /tasks`)](#idempotency-post-tasks)
- [Transactional assignment (`POST /tasks/:id/assign`)](#transactional-assignment-post-tasksidassign)
- [Error format](#error-format)
- [Structured logging](#structured-logging)
- [Testing](#testing)
- [Swagger](#swagger)
- [Postman](#postman)

---

## Tech stack

| Concern         | Choice                                |
|-----------------|---------------------------------------|
| Language        | Go 1.24 (module targets 1.22)         |
| HTTP framework  | Fiber v2                              |
| DB driver       | `jackc/pgx/v5`                        |
| Database        | PostgreSQL 16                         |
| Auth            | JWT (HS256, `golang-jwt/v5`), bcrypt  |
| Logging         | zerolog (structured JSON)             |
| Concurrency     | `golang.org/x/sync/singleflight`      |
| Docs            | Swagger (swaggo) at `/swagger`        |
| API client      | Postman collection (`postman/`)       |
| Container       | Docker + docker-compose               |

---

## Quick start (Docker)

Requires Docker + Docker Compose. One command brings up Postgres and the API (migrations run on startup):

```bash
docker compose up --build
# or: make up   (detached)
```

- API:      http://localhost:3000
- Health:   http://localhost:3000/health
- Swagger:  http://localhost:3000/swagger/index.html

Tear down (including the DB volume):

```bash
docker compose down -v   # or: make down
```

---

## Run locally

Needs Go 1.22+ (a `.go-version` file pins `1.24.0` for goenv users) and a reachable PostgreSQL.

```bash
cp .env.example .env          # adjust DB_* / JWT_SECRET as needed
# start a Postgres (example):
docker run -d --name gdcpay_db -p 5432:5432 \
  -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=gdcpay_tasks \
  postgres:16-alpine

make run        # or: go run ./cmd/api
```
---

## Environment variables

See [`.env.example`](.env.example). Summary:

| Variable           | Default                | Notes                                            |
|--------------------|------------------------|--------------------------------------------------|
| `APP_PORT`         | `3000`                 |                                                  |
| `APP_ENV`          | `development`          | `production` hides internal error detail         |
| `LOG_LEVEL`        | `info`                 | `debug`/`info`/`warn`/`error`                    |
| `DB_HOST`…`DB_NAME`| localhost / postgres…  | PostgreSQL connection                            |
| `JWT_SECRET`       | (change me)            | HS256 signing secret                             |
| `JWT_EXPIRY`       | `24h`                  | access-token lifetime                            |
| `IDEMPOTENCY_TTL`  | `24h`                  | how long an `Idempotency-Key` is remembered      |

## Project structure

```
cmd/api/main.go            entrypoint: config, DB connect, migrate, wire deps, start Fiber
internal/
  config/                  env loading
  domain/                  entities + repository/Store interfaces + AppError (framework-free)
  repository/              raw-SQL (pgx) repos, Store + Atomic, DB connect & schema migrations
  service/                 business logic (auth, task, idempotency, assign, team, comment, notifier)
  handler/                 Fiber handlers, router, swagger annotations
  middleware/              request-id, structured logger, panic recover, JWT auth, error handler
  dto/                     request/response structs + mappers
  pkg/                     jwt, hash (bcrypt), response envelope, validator, logger
test/                      unit tests + in-memory fakes
docs/                      generated Swagger
postman/                   Postman collection (importable)
Dockerfile, docker-compose.yml, Makefile, .env.example
```

---

## API endpoints

Base path: **`/api/v1`**. All task/team/comment routes require `Authorization: Bearer <token>`.

### Auth
| Method | Path             | Description                  |
|--------|------------------|------------------------------|
| POST   | `/auth/register` | Register a user              |
| POST   | `/auth/login`    | Log in, returns a JWT        |

### Tasks
| Method | Path                    | Description                                              |
|--------|-------------------------|---------------------------------------------------------|
| POST   | `/tasks`                | Create a task — **requires `Idempotency-Key` header**   |
| GET    | `/tasks`                | List tasks (filter `status`, `priority`, search `q`, `assignedTo`, `page`, `limit`, `sort`) |
| GET    | `/tasks/:id`            | Task detail                                             |
| PUT    | `/tasks/:id`            | Update task (all fields optional; set `idTeam` to move it into a team you belong to) |
| DELETE | `/tasks/:id`            | Soft-delete task (cascades to comments; logs preserved) |
| POST   | `/tasks/:id/assign`     | Assign to a team member (single transaction)            |
| GET    | `/tasks/:id/logs`       | Task audit log                                          |
| POST   | `/tasks/:id/comments`   | Add a comment                                           |
| GET    | `/tasks/:id/comments`   | List comments                                           |

### Comments
| Method | Path             | Description                       |
|--------|------------------|-----------------------------------|
| PUT    | `/comments/:id`  | Edit a comment (author only)      |
| DELETE | `/comments/:id`  | Delete a comment (author/owner)   |

### Teams
| Method | Path                            | Description                          |
|--------|---------------------------------|--------------------------------------|
| POST   | `/teams`                        | Create a team (creator = owner)      |
| GET    | `/teams`                        | List teams you own/belong to         |
| GET    | `/teams/:id`                    | Team detail + members                |
| POST   | `/teams/:id/members`            | Add a member (owner only)            |
| GET    | `/teams/:id/members`            | List members                         |
| PUT    | `/teams/:id/members/:userId`    | Update a member's status (owner)     |
| DELETE | `/teams/:id/members/:userId`    | Remove a member (owner)              |

### Infra
| Method | Path          | Description |
|--------|---------------|-------------|
| GET    | `/health`     | Liveness    |
| GET    | `/swagger/*`  | Swagger UI  |

---

## Authentication

```bash
# 1) Register
curl -s localhost:3000/api/v1/auth/register -H 'Content-Type: application/json' -d '{
  "name":"Nowaf","username":"nowaf","email":"nowaf@example.com","password":"password123"
}'

# 2) Login -> token
TOKEN=$(curl -s localhost:3000/api/v1/auth/login -H 'Content-Type: application/json' \
  -d '{"login":"nowaf","password":"password123"}' | jq -r .data.token)

# 3) Use it
curl -s localhost:3000/api/v1/tasks -H "Authorization: Bearer $TOKEN"
```

Passwords are bcrypt-hashed and never returned. Tokens are HS256, carrying the user id + username.

---

## Idempotency (`POST /tasks`)

`POST /tasks` **requires** an `Idempotency-Key: <uuid>` header. Replaying the same key (same user,
same body) within `IDEMPOTENCY_TTL` returns the **original** response and creates **no** new task.

```bash
KEY=$(uuidgen)
curl -s localhost:3000/api/v1/tasks \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -H "Idempotency-Key: $KEY" \
  -d '{"title":"Write report","priority":"HIGH"}'

# Same key again -> identical response, no duplicate (header Idempotent-Replayed: true)
curl -s localhost:3000/api/v1/tasks \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -H "Idempotency-Key: $KEY" \
  -d '{"title":"Write report","priority":"HIGH"}'
```

Behaviors:

| Situation                                   | Result                                      |
|---------------------------------------------|---------------------------------------------|
| missing header                              | `400 IDEMPOTENCY_KEY_REQUIRED`              |
| non-UUID header                             | `400 IDEMPOTENCY_KEY_INVALID`               |
| same key, **same** body (completed)         | replay original response, no new task       |
| same key, **different** body                | `409 IDEMPOTENCY_KEY_REUSED`                |
| same key, still being processed (other proc)| `409 IDEMPOTENCY_IN_PROGRESS`               |

**Two layers of defense** guarantee exactly one task per key:

1. **In-process** — `singleflight` collapses concurrent requests with the same key so the work runs
   once; the rest await and receive the same result. (This is what the no-DB race test exercises.)
2. **Cross-process / durable** — a unique index `(user_id, idempotency_key)` plus a single
   transaction: `INSERT … ON CONFLICT DO NOTHING`, create the task, store the response, commit. Only
   one INSERT can win; others read the stored record. The key is scoped per user and a `request_hash`
   detects body mismatches.

---

## Transactional assignment (`POST /tasks/:id/assign`)

Assigning a task to another **active member of the same team** runs in a **single transaction**:

1. lock + load the task,
2. validate the target is an active team member,
3. update `assigned_to`,
4. write a `TaskLog` (`ASSIGN_USER`),
5. dispatch a notification (mock — logged via zerolog).

If **any** step fails (including the notification), the whole transaction **rolls back** — the
assignee and audit log are left untouched. This is covered by a unit test that injects a failing
notifier and asserts nothing was persisted.

```bash
curl -s -X POST localhost:3000/api/v1/tasks/$TASK_ID/assign \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d "{\"assignedTo\":\"$MEMBER_USER_ID\"}"
```

---

## Error format

Every error uses one consistent envelope (`status`, `code`, `message`, `timestamp`):

```json
{
  "status": "error",
  "code": "TASK_NOT_FOUND",
  "message": "task not found",
  "timestamp": "2026-06-13T10:00:00Z"
}
```

- 4xx vs 5xx are clearly separated; validation errors include a `details` array.
- A global error handler maps `AppError`s to the envelope; a recover middleware turns **panics** into
  a generic `500 INTERNAL_ERROR` (no stack traces or internals leak, especially when `APP_ENV=production`).

---

## Structured logging

Every request emits one JSON line. Level is **INFO** for `<400`, **WARN** for `4xx`, **ERROR** for `5xx`.

```json
{"level":"info","service":"gdcpay-task-api","request_id":"6f1c…","method":"POST",
 "path":"/api/v1/tasks","status":201,"latency_ms":12,"user_id":"…","time":"2026-06-13T10:00:00Z","message":"request"}
```

A `request_id` (UUID) is generated per request (honoring an inbound `X-Request-ID`), echoed in the
`X-Request-ID` response header, and attached to every log line for that request.

---

## Testing

Unit tests run **without a database or network** (in-memory `Store` fake). The headline tests prove
there is no race condition on idempotency:

```bash
make test-race      # go test ./test/... -race -count=1 -v
```

- `TestIdempotency_Sequential` — second request with the same key creates no new task; identical body.
- `TestIdempotency_ConcurrentDuplicate` — 50 goroutines, same key, **exactly one** task created.
- `TestInsertIfAbsent_ConcurrentExactlyOnce` — 100 goroutines, the atomic insert primitive yields one winner.
- `TestIdempotency_ReusedKeyDifferentBody` — same key + different body → conflict.
- `TestAssign_Success` / `TestAssign_RollbackOnNotifierFailure` — transaction commits / rolls back atomically.
- `TestAssign_RejectedWithoutTeam` — assignment validation.

Run everything: `make test`.
---

## Swagger

Served at `/swagger/index.html`. Regenerate from code annotations after changing handlers:

```bash
make swag
```
---

## Postman

Import `postman/GDCPAY-Task-API.postman_collection.json` (Postman → **Import** → **File**), or import
the live OpenAPI spec from a running instance (Postman → **Import** → **Link** →
`http://localhost:3000/swagger/doc.json`).

The collection captures the JWT automatically: run **Auth → Login** (or **Register**) and every other
request inherits the Bearer token. **Create Task** stores the new id into `taskId`; re-sending it
without changes reuses the same `Idempotency-Key` to demonstrate idempotent replay (clear the
`idempotencyKey` collection variable to start a fresh key).

---
