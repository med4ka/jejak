# Jejak

Jejak is a link-in-bio page combined with a URL shortener. Creators get one public profile page (`/u/username`) listing their links, plus short URLs (`/r/code`) with click analytics. Built with Go, PostgreSQL, Redis, and Next.js 14. The repo doubles as a staged architecture learning project (see PRD.md), and runs fully in native mode without Docker.

Runtime details, endpoints, and simplifications: [ARCHITECTURE.md](ARCHITECTURE.md). Database tables: [SCHEMA.md](SCHEMA.md).

## Features

- Short URLs with custom slug (up to 30 characters), tags, expiry, and per-device redirect rules
- Public profile page `/u/{username}` with 11 color themes, social links, avatar upload, and manual link ordering
- Click analytics: clicks by day, breakdown by device and referrer, time-range query, CSV export
- Unique click counter next to the total click counter
- Async click logging in full mode: the redirect pushes an event to a Redis list, a worker flushes it to PostgreSQL; baseline mode writes synchronously
- Session auth: register, login, logout, password and email change, delete account, logout on all devices
- API keys for `POST /api/v1/shorten` (100 requests per minute per key), bulk import, bulk QR download
- Rate limits on auth, shorten, bulk, and export endpoints
- UI in three languages: Indonesian (default), English, German; locale stored in the `NEXT_LOCALE` cookie
- Read/write splitting in full mode: writes and auth go to the primary database, redirect reads go to the replica database when `DATABASE_REPLICA_URL` is set

## Tech Stack

| Layer | Tech | Why |
|---|---|---|
| API | Go 1.26, standard library `net/http` (method-pattern ServeMux) | `"GET /api/..."` routing ships with the standard library; no router dependency |
| Database | PostgreSQL 17 | Primary plus a second database that simulates a read replica; schema lives in `backend/db/migrations` |
| Cache and queue | Redis | Cache-aside reads, list queue for click events, lock so only one worker runs |
| Frontend | Next.js 14 (App Router), React 18, Tailwind CSS, Framer Motion | Server-rendered public pages, client-side dashboard |
| i18n | JSON dictionaries in `frontend/messages` | Three flat files, no i18n library; locale resolved from a cookie |
| Local run | Native processes | Docker is unavailable in the dev environment; `infra/docker-compose.yml` is kept as a reference only (ARCHITECTURE.md) |

## Getting Started

### Prerequisites

- Go 1.26+ (`go version`)
- Node.js 18.17+ (Next.js 14 minimum; developed on Node 24)
- PostgreSQL 17
- Redis (local `redis://localhost:6379` or Upstash `rediss://`)

### Setup

All commands below are PowerShell.

```powershell
git clone <repo-url> jejak
cd jejak
Copy-Item .env.example .env          # defaults already point at localhost

# Create the databases (adjust user and path to your install)
& "C:\Program Files\PostgreSQL\17\bin\createdb.exe" -U postgres jejak
& "C:\Program Files\PostgreSQL\17\bin\createdb.exe" -U postgres jejak_replica   # needed for full mode reads

# Migrations: the primary migrates on first server start.
# The replica is not touched at startup, sync it explicitly (idempotent):
cd backend
go run ./cmd/migrate-replica
cd ..

# Optional: seed test rows (short code abc123 plus an owner)
& "C:\Program Files\PostgreSQL\17\bin\psql.exe" -U postgres -d jejak -f backend/db/seed.sql
& "C:\Program Files\PostgreSQL\17\bin\psql.exe" -U postgres -d jejak_replica -f backend/db/seed.sql

cd frontend
npm install
cd ..
```

Key `.env` values: `APP_MODE` (`baseline`, `full`, or `shard`), `DATABASE_URL`, `DATABASE_REPLICA_URL` (empty = no replica), `REDIS_URL` (empty = no cache and synchronous click logging), `GO_API_URL` (frontend to API origin).

### Running (3 terminals)

```powershell
# Terminal 1: API (from backend/). PORT defaults to 8080; 8081 matches the frontend default.
cd backend
$env:PORT = "8081"; $env:APP_MODE = "full"; go run ./cmd/server

# Terminal 2: worker (from backend/). Needs Redis; exits if Redis is down
# or another worker holds the lock.
cd backend
go run ./cmd/worker

# Terminal 3: frontend (from frontend/)
cd frontend
npm run dev                          # http://localhost:3000
```

Notes:

- The frontend proxies `/api/*`, `/uploads`, and `/r/{code}` to `GO_API_URL` (default `http://localhost:8081`). If the API runs on another port, set `$env:GO_API_URL` before `npm run dev` or `npm run build`.
- If port 8081 is taken on your machine (for example by another local web server), run the API on `8082` and point `GO_API_URL` at it.
- Baseline run without cache, replica, or queue: `$env:APP_MODE = "baseline"; $env:PORT = "8081"; go run ./cmd/server`.
- Do not run `npm run dev` and `npm run start` at the same time: both use the `.next` folder. Stop both and run `npm run clean` if routes start returning 500.

## Development

### Project Structure

```
jejak/
├── backend/
│   ├── cmd/                 # server, worker, proxy (round-robin LB), loadtest, smoketest, migrate-replica
│   ├── internal/            # handler, auth, cache, db, env, middleware, migrate, ratelimit, shortener
│   └── db/                  # migrations (14 up, 12 down) + seed.sql
├── frontend/
│   ├── app/                 # routes: /, /dashboard, /u/[username], /r, /login, /register, ...
│   ├── components/          # React components
│   ├── lib/                 # i18n.js, themes.js
│   └── messages/            # id.json, en.json, de.json
├── infra/                   # docker-compose.yml + nginx.conf (reference only, not run locally)
├── README.md, ARCHITECTURE.md, SCHEMA.md
└── PRD.md, RULES.md, DESIGN.md, PROGRESS.md
```

### Smoke Test

12 checks: register, login, profile, shorten, redirect, analytics, update profile, update email, delete account, 401 after delete. Exit code 0 = all pass, 1 = failure (CI-ready). The test account is random per run and deleted at the end.

```powershell
cd backend
go run ./cmd/smoketest                              # default base URL http://localhost:8081
$env:BASE_URL = "http://localhost:8082"; go run ./cmd/smoketest
$env:SMOKE_STRICT = "1"; go run ./cmd/smoketest     # strict replica assertions
```

### i18n

To add a language:

1. Copy `frontend/messages/en.json` to `frontend/messages/<code>.json` and translate the values (keys must stay identical).
2. Add `<code>` to `LOCALES` in `frontend/lib/i18n.js`.

The switcher renders from `LOCALES`; the choice is stored in the `NEXT_LOCALE` cookie, default `id`.

### Themes

The 11 presets: `classic`, `darkroom`, `coral`, `glass`, `risoPrint`, `peach`, `lavender`, `matcha`, `sakura`, `ocean`, `sunset`. Themes apply to `/u/{username}` only (plus the navbar while that page is open).

To add one:

1. Add the preset class mapping in `frontend/lib/themes.js` (the single source of truth for the frontend).
2. Add the key to `validThemePreset` in `backend/internal/handler/handlers_profile.go` (the API rejects unknown themes with 400).
3. Optional: add theme-specific styles under `body[data-profile-theme]` in `frontend/app/globals.css`.

## Deployment

Guide: `docs/deploy.md` (placeholder, not written yet). Short version: set `DATABASE_URL`, `REDIS_URL`, `APP_MODE`, `PORT` for the API; build with `go build -o jejak-server ./cmd/server`; build the frontend with `npm run build` and serve it with `npm run start` behind `GO_API_URL` pointing at the API.

## License

No license file in this repository yet.
