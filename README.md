# Life Dashboard Server

This is a backend service built in Go to interact with the Google ecosystem, allowing users to access and manipulate their data across different Google services. Built with industry best practices in mind, it utilizes an onion architecture with Data Transfer Objects (DTOs) to maintain a strict separation of concerns.

## Features

* **Google Auth Integration:** Uses OAuth2 and the Google login flow for secure authentication, storing user credentials in a MySQL database. All OAuth tokens are AES-256 encrypted before persistence.
* **Task Management:** Two-way synchronization # Life Dashboard Server

The Go backend for Life Dashboard. It wraps the Google ecosystem (Tasks, Calendar) behind a single authenticated REST API, and adds its own domain concepts on top — routines, time zones, notes and a daily scratchpad — that Google doesn't provide.

Built with an onion architecture and DTOs at every boundary, so the business logic never touches HTTP or SQL directly.

**Module:** `github.com/nickquirk/life-dashboard-server`

## Features

**Google OAuth2 login.** Full authorization-code flow with a CSRF state cookie. Requests the `tasks`, `userinfo.email` and `calendar.readonly` scopes. Google access and refresh tokens are AES-256-GCM encrypted before they ever hit the database.

**Task management.** Two-way sync with Google Tasks — create, update, move, batch-delete tasks and subtasks, plus local subtask reordering that Google's API doesn't support natively.

**Calendar events.** Read-only Google Calendar access over an explicit start/end window (RFC3339).

**Routines.** Recurring activity templates with optional goals (time-based or count-based) and reset periods (one-off, weekly, monthly). Individual instances are scheduled onto the calendar and tracked separately from the template.

**Zones.** Daily time blocks with a schedule, active weekdays and a colour — the "sleep / work / lunch" backdrop the calendar renders against.

**Notes.** Three note types (text, checklist, bullet) with orderable items, pinning and archiving.

**Scratchpad.** A single free-text pad per user, upserted rather than versioned.

**Sessions.** Short-lived JWT session cookie (6 hours) paired with a 30-day refresh cookie. Refresh tokens are stored SHA-256 hashed and rotated on every use — the old session row is deleted before the new one is written, so a leaked token can't be replayed. Multiple devices are supported and capped per user.

## Tech Stack

| | |
|---|---|
| Language | Go 1.25 |
| Routing | Chi v5 (+ `go-chi/cors`) |
| Database | MySQL via GORM |
| DI | Uber dig |
| Config | Koanf (YAML) + godotenv (`.env`) |
| Auth | `golang-jwt/jwt/v5`, `golang.org/x/oauth2` |
| Secrets | GCP Secret Manager (prod) |
| Logging | `log/slog`, JSON handler |
| Testing | `stretchr/testify`, hand-written mocks, SQLite for repository tests |

## Architecture

Dependencies point inward. Handlers know about services; services know about repositories; nothing points back out.

```
cmd/http/              Entry point — loads .env, builds the container, runs
internal/
  config/              Koanf loader + Google OAuth2 config
  container/           dig wiring (container.go) and lifecycle (application.go)
  crypto/              AES-256-GCM token encryptor + key loading
  db/                  Connection, pooling, AutoMigrate + one-off data migrations
  domain/              GORM models, DTOs, validation, custom JSON types
  repository/          Data persistence — the only layer that touches GORM
  service/             Business logic and Google API communication
  handlers/            HTTP handlers, routing, middleware, cookie config
  testutil/mocks/      Interface mocks with overridable function fields
  utils/               JWT generation, verification, refresh token hashing
```

A couple of details worth knowing:

- **`domain` carries its own validation.** `Validate()` methods on request DTOs, GORM hooks in `routines_hooks.go`, and custom unmarshalling in `nullable_date.go` and `goal_update.go` to distinguish "field absent" from "field explicitly cleared" in PATCH payloads.
- **Mocks use nil-checked function fields.** Set only the method you care about; everything else returns zero values.
- **Errors are sanitised at the boundary.** `respondWithError` logs the real error with structured context and returns a generic message to the client.

## Configuration

Config comes from two places: a YAML file for non-secret app settings, and environment variables for everything else.

The YAML path is `CONFIG_PATH`, defaulting to `config.dev.yaml`. It must define `app.name`. Locally, a missing file panics — deliberately, so you find out immediately. In production a missing file only warns, since Cloud Run supplies everything via env vars.

Create a `.env` in the project root for local secrets (see `.env.example`):

| Variable | Required | Notes |
|---|---|---|
| `ENV` | | `prod` switches off `.env` loading, startup migrations and dev-only behaviour |
| `CONFIG_PATH` | | Defaults to `config.dev.yaml` |
| `PORT` | | Falls back to `service.port` from the YAML config |
| `DB_CONNECTION` | Yes | Only `mysql` is supported |
| `DB_USER` / `DB_PASS` / `DB_NAME` | Yes | |
| `DB_HOST` | | TCP connection (local / private IP) |
| `DB_SOCKET` | | Unix socket — takes precedence over `DB_HOST`, used by Cloud SQL |
| `JWT_SECRET` | Yes | Missing value exits the process rather than silently issuing 401s |
| `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET` | Yes | |
| `REDIRECT_URL` | Yes | Must match the OAuth callback registered with Google |
| `CLIENT_URL` | Yes | Sole allowed CORS origin; startup fails without it |
| `TOKEN_ENCRYPTION_KEY` | Yes | Base64-encoded, must decode to exactly 32 bytes |
| `CROSS_ORIGIN` | | `true` sets `SameSite=None` (needed when FE and BE are separate origins) |
| `COOKIE_DOMAIN` | | e.g. `.example.com` for shared subdomain cookies |
| `MIGRATE_ONLY` | | `true` runs migrations and exits without starting the server |
| `CLOUD_RUN_URL` | | OIDC audience; when unset, OIDC middleware is a no-op |

### Encryption key

`LoadEncryptionKey` checks `TOKEN_ENCRYPTION_KEY` first. If it's unset and `ENV=prod`, it falls back to GCP Secret Manager (`token-encryption-key`), accepting either a base64 payload or raw 32 bytes. Outside prod with no env var set, startup fails.

To generate one:

```bash
openssl rand -base64 32
```

### Cookies

`NewCookieConfig` resolves settings from the environment at startup:

- `ENV=prod` → `Secure=true`
- `CROSS_ORIGIN=true` → `SameSite=None` **and** `Secure=true` (browsers require it)
- Otherwise → `SameSite=Lax`, insecure, suitable for `localhost`

Three cookies are issued: `life-dashboard` (JWT, 30-day max-age), `life-dashboard-refresh` (30 days) and `oauthstate` (10 minutes, cleared as soon as it's validated). All are `HttpOnly`.

## Running Locally

```bash
go run ./cmd/http
```

Outside `ENV=prod`, `AutoMigrate` runs on every startup, so a fresh MySQL database will schema itself on first boot.

### Docker

A multi-stage Dockerfile produces a statically linked binary on a minimal Alpine image running as a non-root user.

```bash
docker build -t life-dashboard-server .
docker run -p 8080:8080 --env-file .env life-dashboard-server
```

### Tests

```bash
go test ./...
go test -cover ./...
```

Repository tests run against in-memory SQLite; handler and service tests use the mocks in `internal/testutil/mocks`.

## API

All routes are prefixed `/api` except the health checks. Authenticated routes require the `life-dashboard` session cookie; the `Authenticate` middleware validates it and puts the user ID into the request context.

### Public

| Method | Path | |
|---|---|---|
| `GET` | `/health` | Liveness — always 200 |
| `GET` | `/ready` | Readiness — 503 if the DB ping fails |
| `GET` | `/api/auth/google-login` | Redirects to Google's consent screen |
| `GET` | `/api/auth/google-callback` | Exchanges the code, issues cookies |
| `POST` | `/api/auth/logout` | Clears cookies, deletes that one session |
| `POST` | `/api/auth/refresh` | Rotates the JWT and refresh token |

### Authenticated

| Method | Path | |
|---|---|---|
| `GET` | `/api/auth/me` | Current user's email and picture |
| `GET` | `/api/user` | Current user ID |
| `DELETE` | `/api/users/me` | Deletes the account and all local data |
| `GET` | `/api/tasks` | All task lists |
| `POST` | `/api/tasks/sync` | Sync task lists from Google |
| `GET` | `/api/tasks/{taskListId}` | Tasks in a list |
| `POST` | `/api/tasks/{taskListId}` | Create a task |
| `POST` | `/api/tasks/{taskListId}/sync` | Sync tasks in a list |
| `PATCH` | `/api/tasks/{id}` | Update a task |
| `PUT` | `/api/tasks/{id}/subtasks/reorder` | Reorder subtasks |
| `DELETE` | `/api/tasks/{id}` | Delete a task |
| `DELETE` | `/api/tasks/batch` | Delete multiple tasks |
| `GET` | `/api/calendar/events` | Requires `start` and `end` in RFC3339 |
| `GET` `POST` | `/api/zones` | |
| `PATCH` `DELETE` | `/api/zones/{id}` | |
| `GET` `POST` | `/api/routines` | Routine templates |
| `PATCH` `DELETE` | `/api/routines/{id}` | |
| `GET` `POST` | `/api/routines/instances` | `GET` takes `start` and `end` |
| `PATCH` `DELETE` | `/api/routines/instances/{id}` | |
| `GET` `POST` | `/api/notes` | `GET` accepts `?archived=true` |
| `PATCH` `DELETE` | `/api/notes/{id}` | |
| `POST` | `/api/notes/{id}/items` | |
| `PUT` | `/api/notes/{id}/items/reorder` | |
| `PATCH` `DELETE` | `/api/notes/{id}/items/{itemId}` | |
| `GET` `PUT` | `/api/scratchpad` | `PUT` upserts |
| `POST` | `/api/feedback` | |

When Google rejects a stored refresh token, the affected endpoints return `401` with a message telling the client to log in again, rather than a generic `500`.

## Deployment

`cloudbuild.yaml` builds the image, pushes to Artifact Registry, runs migrations as a separate Cloud Run **job**, then deploys the service to Cloud Run in `europe-west2`.

Migrations are deliberately split out. In production, startup migrations are skipped entirely — instead, `backend-migrate` runs the same image with `MIGRATE_ONLY=true`, executes `InitMigration`, and exits. That keeps schema changes off the request path and out of every cold start.

The deployed service uses `/health` as its liveness probe and `/ready` as its startup probe, connects to Cloud SQL over a unix socket, and pulls `DB_PASS`, `JWT_SECRET`, the Google client credentials and `TOKEN_ENCRYPTION_KEY` from Secret Manager.

### Runtime notes

- Connection pool is capped at 5 open / 2 idle connections with a one-hour lifetime — Cloud Run scales horizontally, so per-instance pools have to stay small or Cloud SQL runs out of connections.
- Server timeouts: 15s read, 60s write, 60s idle.
- `SIGINT`/`SIGTERM` trigger a graceful shutdown with a 30-second drain.

### Migration history

`InitMigration` carries two irreversible steps beyond `AutoMigrate`, both idempotent:

1. Drops the deprecated `users.app_refresh_token` column, replaced by the `sessions` table. Existing users must re-authenticate after this deploy.
2. Backfills `routines.target_total_mins` into `goal_type` / `goal_target`, then drops the old column. The backfill panics on failure so the column is never dropped after a partial migration. Google Tasks. Supports creating, updating, moving, and deleting tasks and subtasks directly through the API.
* **Calendar Events:** Read-only access to Google Calendar events based on designated start and end time windows.
* **Time "Zones":** Custom logic for managing daily time blocks (Zones) with specific schedules, days active, and color coding.
* **Secure Sessions:** Issues secure, HTTP-only JWT session cookies with automated, hashed refresh token handling.

## Tech Stack

* **Language:** Go 1.24
* **Routing:** Chi Router
* **Database:** MySQL via GORM
* **Dependency Injection:** Uber Dig
* **Configuration:** Koanf for YAML files and godotenv for `.env` management

## Architecture

The application is heavily structured around the Onion/Clean Architecture pattern to ensure modularity and ease of testing:

* **`cmd/http`**: Application entry point and initialization.

* **`internal/domain`**: Core database models and Data Transfer Objects (DTOs).

* **`internal/repository`**: Data persistence layer interacting with the SQL database.

* **`internal/service`**: Core business logic and external Google API communication.

* **`internal/handlers`**: HTTP handlers, Chi routing, and middleware (including OIDC validation for Cloud Run).

## Setup & Configuration

The server relies on a mix of environment variables and a YAML configuration file. By default, it looks for `config.dev.yaml` locally.

Create a `.env` file in the root directory for your secrets (see `.env.example`  for an example).

## Running the Application Locally

```Bash
go run ./cmd/http
Using Docker:
A multi-stage Dockerfile is included for generating a minimal, statically linked Alpine container.
```

```Bash
docker build -t life-dashboard-server .
docker run -p 8080:8080 --env-file .env life-dashboard-server
```
