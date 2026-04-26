# YouTube Share

A small full-stack app for sharing YouTube videos with friends. Sign up, paste a link, and every other signed-in user gets a real-time toast announcing the new share — courtesy of a WebSocket fan-out driven by an in-process background worker on the Go backend.

This is a monorepo with two packages:

```
youtube-share/
├── backend/    # Go 1.22 + chi + GORM + PostgreSQL + Redis + WebSocket
├── frontend/   # React 19 + Vite + TanStack Router (file-based) + Tailwind 4 (pnpm)
├── docker-compose.yml
└── README.md
```

The frontend uses **pnpm** as its package manager — pinned in `frontend/package.json` via the `packageManager` field, so Corepack provisions the exact version on every machine. `pnpm-lock.yaml` is the only lockfile we commit; `yarn.lock` and `package-lock.json` are ignored.

## 1. Introduction

**Purpose.** Demonstrate end-to-end fullstack engineering: typed APIs, modular architecture, real-time notifications, caching, background jobs, automated tests and Dockerized deployment.

**Key features.**

- Email/password registration and sign-in with JWT (access + refresh).
- Paste any YouTube URL (watch, `youtu.be`, embed, shorts) — the backend extracts the canonical video id and stores it.
- Feed of recently shared videos with thumbnail and sharer info.
- Real-time **video_shared** notification broadcast over WebSockets to every other signed-in user.
- Redis cache for the recent-videos list (30s TTL, invalidated on share).
- Background worker that fans the WebSocket broadcast off the request goroutine.
- i18n (English + Vietnamese) on the frontend.
- Unit tests (Vitest, Go `testing`) and an end-to-end integration test that exercises the sign-up → share → WebSocket-notify flow.

## 2. Prerequisites

| Tool | Version |
|------|---------|
| Go | 1.22+ |
| Node | 20+ |
| pnpm | 9+ (`corepack enable` — the version is pinned via `packageManager` in `frontend/package.json`) |
| PostgreSQL | 14+ (16 recommended) |
| Redis | 7+ |
| Docker + Docker Compose | latest |
| `golangci-lint` | 1.61+ (optional, for `make lint`) |
| `gofumpt`, `goimports` | latest (optional, for `make fmt`) |

The Docker workflow only needs Docker. Everything else is for running the apps natively.

## 3. Installation & Configuration

```bash
git clone <repo-url> youtube-share
cd youtube-share
```

### Backend

```bash
cd backend
cp .env.example .env             # adjust DB / Redis / JWT secrets
go mod tidy
```

Environment variables (`backend/.env`):

| Name | Default | Notes |
|------|---------|-------|
| `HTTP_PORT` | `8080` | API listen port |
| `DB_HOST` / `DB_PORT` / `DB_USER` / `DB_PASSWORD` / `DB_NAME` / `DB_SSLMODE` | postgres locals | PostgreSQL DSN parts |
| `REDIS_ADDR` | `localhost:6379` | host:port |
| `JWT_ACCESS_SECRET` / `JWT_REFRESH_SECRET` | dev-only | **change for prod** |
| `JWT_ACCESS_TTL_MIN` | `15` | access token lifetime |
| `JWT_REFRESH_TTL_HOURS` | `168` | refresh token lifetime |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:5173,http://localhost:3000` | comma-separated |
| `MIGRATIONS_DIR` | `./migrations` | path to versioned `*.up.sql` / `*.down.sql` files (read by `cmd/migrate`) |

### Frontend

```bash
cd frontend
cp .env.example .env             # API base URLs default to http://localhost:8080

# One-time on a fresh machine — Corepack ships with Node 16.10+ and
# auto-provisions the pnpm version pinned in package.json#packageManager.
corepack enable

pnpm install                     # produces pnpm-lock.yaml + node_modules
```

> **Why pnpm?** Faster installs, content-addressed store (no per-project duplication), strict peer-dep checks. The only quirk is that strict peer mode hides indirect deps from sub-packages; we work around that with `frontend/.npmrc`:
>
> ```ini
> auto-install-peers=true     # avoids peer-warning install failures
> shamefully-hoist=true       # makes things like `react` visible to plugins that read from node_modules root
> ```

Environment variables (`frontend/.env`):

| Name | Default | Notes |
|------|---------|-------|
| `VITE_API_BASE_URL` | `http://localhost:8080/api/v1` | REST endpoint |
| `VITE_WS_BASE_URL` | `ws://localhost:8080/api/v1` | WebSocket endpoint |

## 4. Database Setup

For day-to-day development the recommended workflow is **infrastructure in Docker, app code on the host** — that way you get hot reload from `go run` / `pnpm dev` while Postgres and Redis run in containers.

A dedicated compose file at the repo root spins up only the dependencies:

```bash
docker compose -f docker-compose.dev.yml up -d        # start Postgres + Redis
docker compose -f docker-compose.dev.yml ps           # status
docker compose -f docker-compose.dev.yml logs -f      # tail logs
docker compose -f docker-compose.dev.yml down         # stop (keep data)
docker compose -f docker-compose.dev.yml down -v      # stop + wipe data
```

There are Make shortcuts inside `backend/`:

```bash
cd backend
make dev      # one-shot: start Postgres+Redis, then run the API
make up       # start Postgres+Redis only (background)
make down     # stop containers, keep data
make reset    # stop containers and DELETE data (fresh DB)
```

If you'd rather run them ad-hoc without the compose file:

```bash
docker run --name ytshare-postgres -e POSTGRES_USER=ytshare \
  -e POSTGRES_PASSWORD=ytshare -e POSTGRES_DB=ytshare \
  -p 5432:5432 -d postgres:16-alpine

docker run --name ytshare-redis -p 6379:6379 -d redis:7-alpine
```

The full-stack `docker-compose.yml` (frontend + backend + DB + Redis + one-shot migrate job) is the **deploy-style** run; use it to verify the production image builds end-to-end. For everyday coding, prefer `docker-compose.dev.yml`.

### Migrations (Flyway-style)

Schema changes are managed by **[`golang-migrate`](https://github.com/golang-migrate/migrate)** — the Go-native equivalent of Flyway. Migrations are versioned, immutable SQL files under `backend/migrations/`:

```
backend/migrations/
├── 000001_init.up.sql
├── 000001_init.down.sql
└── README.md
```

Naming: `<6-digit-sequence>_<short_name>.{up,down}.sql`. Once a migration has been applied to any environment, **never edit it** — create a new one. The migrator records applied versions in a `schema_migrations` table inside the same database.

The server never auto-applies migrations — that's an explicit operator step so the API can never silently mutate the database. Apply migrations the first time and after every schema change:

```bash
cd backend
make migrate                     # apply all pending migrations
make migrate-new name=add_votes  # scaffold the next *.up.sql / *.down.sql pair

# Less common — call the CLI directly:
go run ./cmd/migrate down            # roll back the most recent migration
go run ./cmd/migrate status          # print current schema version
go run ./cmd/migrate force <N>       # mark a version applied (recovery only)
```

Both Make targets shell out to `go run ./cmd/migrate …`, which reads the same `.env` as the server. The Docker image ships with `/app/migrate` alongside `/app/server`, and `docker-compose.yml` runs a one-shot `migrate` service before the backend starts (no on-boot migration in the API container).

There is no seed data — sign up a couple of accounts and start sharing.

## 5. Running the Application

### Backend

```bash
cd backend

# First time only — apply schema after Postgres comes up.
make up && make migrate

# Day-to-day:
make dev                  # bring up Postgres+Redis, then run the API
```

`make dev` shells out to `docker compose -f docker-compose.dev.yml up -d --wait` and then `go run ./cmd/server`. The server never auto-mutates the DB — run `make migrate` explicitly whenever you add a new migration. Ctrl+C stops the API; `make down` stops the containers when you're done.

The API listens on `http://localhost:8080`. A health probe is exposed at `GET /healthz`.

Other targets (run `make help` for the full list):

```bash
make test            # unit + integration tests, race detector on
make build           # build server + migrate binaries into ./bin
make lint            # golangci-lint, Uber-Go-style config
make fmt             # gofumpt + goimports
make tidy            # go mod tidy
make migrate         # apply pending migrations explicitly
make migrate-new name=add_votes
```

### Frontend

```bash
cd frontend
pnpm dev                  # http://localhost:5173
```

All scripts (defined in `frontend/package.json#scripts`):

| Command | What it does |
|---------|--------------|
| `pnpm dev` | Vite dev server with HMR + TanStack Router file-watching |
| `pnpm build` | Type-check + production bundle into `dist/` |
| `pnpm preview` | Serve the production build locally |
| `pnpm typecheck` | `tsc --noEmit` against `tsconfig.app.json` |
| `pnpm lint` / `pnpm lint:fix` | ESLint flat config |
| `pnpm format` / `pnpm format:check` | Prettier |
| `pnpm test` / `pnpm test:ui` | Vitest (jsdom) — unit + component tests |
| `pnpm test:e2e` | Playwright (configure in `playwright.config.ts` first) |

The TanStack Router file-based plugin generates `src/routeTree.gen.ts` automatically when the dev server starts.

## 6. Docker Deployment

A single `docker-compose.yml` at the repo root spins up Postgres, Redis, the Go backend (multi-stage `distroless` image), and the React frontend (multi-stage build that runs **`pnpm install --frozen-lockfile`** + `pnpm build` via Corepack, served by nginx).

```bash
docker compose up --build
```

Then open:

- Frontend: <http://localhost:3000>
- Backend (API): <http://localhost:8080/api/v1>
- Health: <http://localhost:8080/healthz>

To rebuild a single service:

```bash
docker compose build backend
docker compose up backend
```

Stop and remove containers (keeping the volume):

```bash
docker compose down
```

Reset the database too:

```bash
docker compose down -v
```

## 7. Usage

1. Open the frontend, click **Sign up** and create an account.
2. Open a second browser (or incognito tab), sign up as a different user.
3. In one window go to **Share** in the top nav, paste any YouTube URL and submit.
4. The other window receives a toast — *"Alice shared: …"* — and the videos list refreshes automatically.
5. Click any card to open the video on YouTube.

API surface:

| Method | Path | Auth | Purpose |
|--------|------|------|---------|
| `POST` | `/api/v1/auth/signup` | — | Create account, returns JWT pair |
| `POST` | `/api/v1/auth/signin` | — | Sign in, returns JWT pair |
| `POST` | `/api/v1/auth/refresh` | — | Exchange refresh token for new pair |
| `GET`  | `/api/v1/videos` | — | List recently shared videos (30s Redis cache) |
| `POST` | `/api/v1/videos` | Bearer | Share a video |
| `GET`  | `/api/v1/notifications/ws` | Bearer (header or `?access_token=`) | WebSocket subscribe |

Project structure highlights:

```
backend/
├── cmd/server/                # main()
└── internal/
    ├── config/                # env loader + validation
    ├── database/              # postgres + redis + auto-migrate
    ├── cache/                 # Cache interface + Redis + in-memory impls
    ├── jobs/                  # background worker pool
    ├── logger/                # slog JSON logger
    ├── middleware/            # cors, request logger, auth
    ├── modules/
    │   ├── auth/              # jwt issuer, signup/signin/refresh, handlers
    │   ├── users/             # User model + repository
    │   ├── videos/            # share + list, YouTube URL parsing, cache
    │   └── notifications/     # WebSocket hub + client + handler
    └── server/                # router wiring + graceful shutdown

frontend/src/
├── components/ui/             # primitives (Button, Input, Card, …)
├── shared/{components,constants,utils,components/Form}
├── modules/
│   ├── auth/                  # signin/signup forms + pages, store, services
│   ├── videos/                # share form, video list/card, services, hooks
│   └── notifications/         # WebSocket subscription hook
└── routes/                    # TanStack file-based: _public/, _private/
```

## 8. Troubleshooting

**`connection refused` on backend startup.** Postgres or Redis isn't reachable. Confirm they're running on the host/port in `.env` and that `pg_isready` / `redis-cli ping` succeed.

**`pq: role "ytshare" does not exist`.** Recreate the postgres container with the env vars from `docker-compose.yml`, or `CREATE ROLE ytshare LOGIN PASSWORD 'ytshare';` then `CREATE DATABASE ytshare OWNER ytshare;`.

**Frontend shows blank page after sign-in.** Make sure `VITE_API_BASE_URL` (and `VITE_WS_BASE_URL` for live notifications) point at a reachable backend. The dev server defaults to `http://localhost:8080/api/v1`.

**`pnpm: command not found`.** Run `corepack enable` once. Corepack ships with Node 16.10+ and provisions pnpm at the version pinned in `package.json`'s `packageManager` field. If your Node is older, install pnpm directly: `npm install -g pnpm@9`.

**Strict peer-dep errors during `pnpm install`.** Already mitigated by `frontend/.npmrc` (`auto-install-peers=true`, `shamefully-hoist=true`). If you bumped React or a TanStack package and see new warnings, re-run with `pnpm install --no-strict-peer-dependencies` once to confirm before pinning the resolution.

**`Cannot find module '@tanstack/react-router'`** (or any other dep) in your editor. Three possibilities, in order of likelihood:

1. The TS server is stale — in VS Code / Cursor: `⌘⇧P` → **TypeScript: Restart TS Server**.
2. The install completed in the wrong directory. Confirm `pwd` prints `/path/to/youtube-share/frontend` and `ls node_modules/@tanstack/react-router/package.json` exists.
3. The editor opened the monorepo root and is using the wrong tsconfig. Either open `frontend/` directly, or set `"typescript.tsdk": "frontend/node_modules/typescript/lib"` in `.vscode/settings.json`.

**`yarn.lock` / `package-lock.json` showed up after a stray install.** Both are gitignored, but delete them locally (`rm yarn.lock package-lock.json`) and rerun `pnpm install` so the project stays single-lockfile.

**`401 Unauthorized` after a while.** Access tokens expire every 15 minutes by default. The axios interceptor uses the refresh token automatically, but the refresh token itself expires after 7 days — sign in again.

**WebSocket disconnects every minute.** That's the ping/pong keepalive doing its job; the client reconnects automatically. If you see *every connect* fail, check `CORS_ALLOWED_ORIGINS` includes the frontend origin.

**Compose says `port is already in use`.** Stop whatever is holding 5432 / 6379 / 8080 / 3000 (`lsof -i :PORT`) or remap the published port in `docker-compose.yml`.

**`youtube_id` column not found after upgrade.** The schema now lives in `backend/migrations/000001_init.up.sql`. If you upgraded from a build that used GORM's `AutoMigrate` (`you_tube_id`), drop the legacy tables once and let the SQL migration recreate them: `psql … -c 'DROP TABLE IF EXISTS videos, users CASCADE;'` then `make migrate-up`.

**`Dirty database version N. Fix and force version.`** A migration crashed mid-way. Open `migrations/000XXX_*.up.sql`, fix what broke (or restore the DB from a backup), then mark the version applied with `make migrate-force version=N` and re-run `make migrate-up`.

**Tests fail with `gofumpt` / `goimports` not found.** Install them once:

```bash
go install mvdan.cc/gofumpt@latest
go install golang.org/x/tools/cmd/goimports@latest
```

---

Built following the [Uber Go style guide](https://github.com/uber-go/guide/blob/master/style.md) on the backend.
