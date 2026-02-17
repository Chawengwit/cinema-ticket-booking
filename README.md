# Cinema Ticket Booking Platform

Cinema Ticket Booking is a full-stack reference implementation for a high-concurrency movie ticketing experience. It demonstrates how to combine Go (Gin) + MongoDB + Redis with a Vue 3 frontend, Google OAuth, and Docker-first DevOps practices while satisfying the requirements for distributed locking, auditability, and real-time seat visibility.

## Feb 17, 2026 Review Snapshot
### Implemented Today
- Google OAuth login + callback persist Google profiles in Mongo, mint JWTs, and expose `/api/me` + `/api/admin/ping` so the Vue shell can show who is signed in.
- Seat locking service uses Redis Lua scripts for atomic multi-seat locks, TTL refreshes, release helpers, and HTTP handlers (`POST/DELETE /api/showtimes/:id/seats/lock`) that enforce authentication.
- Docker Compose + Vite proxy provide a one-command environment (`docker compose up --build`) that wires Mongo, Redis, the Go API, and the Vue dev server together with hot reload.
- Vue SPA handles OAuth callback tokens, renders login/logout state, and calls the backend health endpoint so people can verify the stack quickly.

### High-priority follow-ups
- Secrets (`JWT_SECRET`, `GOOGLE_CLIENT_*`) are committed in `.env`; replace with `.env.example`, rotate credentials, and keep real values outside version control.
- `backend/internal/http/handler/auth_google.go` still sends a fixed `"dev-state"` and never validates the `state`/`error` parameters or uses PKCE/nonce protection, so the OAuth flow is vulnerable to CSRF and replay.
- `GET /api/showtimes/:id/seats/locks` is a debug endpoint but is accessible to any authenticated user and currently leaks lock owners/TTL; restrict it (admin-only) or guard behind build tags before production.
- No automated tests exist for seat locking, OAuth, or repositories; add at least table-driven unit tests for `seatlock.Service` Lua flows and handler-level tests for `/api/me`/seat lock endpoints.
- README previously referenced `docs/TASK_01_DOMAIN_MODELS.md`, but that file is not in the repo; either add the doc or adjust references when the domain spec lands.

## Requirements Snapshot
- **User booking flow**: Google OAuth sign-in → seat selection → 5-minute Redis distributed lock → payment or timeout → final booking persisted in MongoDB.
- **Real-time UX**: Seat map status (AVAILABLE / LOCKED / BOOKED) pushed live over WebSocket/SSE and mirrored through Redis Pub/Sub so every client sees conflicts immediately.
- **Admin ops**: Dashboard for booking listings + filters, role-gated APIs, and audit log stream for booking success/timeout/seat release/system errors.
- **Non-functional guardrails**: Zero double-booking via Redis locks, asynchronous event bus for notifications/logging, env-driven configuration, and single-command deployment using `docker compose up --build`.

## Repository Layout
```
.
├── backend              # Go services (Gin API, Mongo, Redis clients)
│   ├── cmd/api          # HTTP entrypoint
│   └── internal         # Application packages
│       ├── auth         # JWT service
│       ├── cache        # Redis connection setup
│       ├── config       # Env configuration
│       ├── db           # Mongo connection setup
│       ├── http         # Handlers + middleware
│       ├── model        # Core business entities (User for now)
│       ├── repo         # MongoDB repositories
│       └── seatlock     # Redis-based locking service
├── frontend             # Vue 3 + Vite SPA (seat map, admin console)
├── docker-compose.yml   # Orchestrates mongo, redis, backend, frontend
└── .env                 # Local configuration (never commit secrets)
```

## Tech Stack & Responsibilities
| Layer | Technology | Purpose |
| --- | --- | --- |
| API Gateway | Go 1.24 + Gin | REST APIs, WebSocket/SSE channel, seat lock orchestration |
| Persistence | MongoDB 7 | Stores movies, showtimes, seat maps, bookings, audit logs |
| Cache / Locking | Redis 7 | 5-minute distributed locks, hot seat map cache, Pub/Sub fan-out |
| Real-time Bus | Redis Pub/Sub | Broadcasts seat lock/unlock + booking success to API nodes |
| Frontend | Vue 3 + Vite + Tailwind | User booking flow, admin dashboard, WebSocket client |
| Auth | Google OAuth 2.0 + JWT | Federated login, role claims (USER / ADMIN) |
| Messaging | Redis Pub/Sub → Notification worker (mock) | Fire-and-forget notifications + async logging |
| Containers | Docker + docker-compose | Single command local deployment + parity with CI |

## Current Implementation Status (Feb 17, 2026)
- ✅ Docker Compose boot plus `/health` verification covers Mongo, Redis, and Gin with a lean multi-stage backend image.
- ✅ Google OAuth login/callback mints JWTs, stores users in Mongo, powers `/api/me`, and guards `/api/admin/ping` via role middleware.
- ✅ Seat locking lives in Redis with Lua-based all-or-nothing locks, request-id echoing, release helpers, and authenticated HTTP handlers.
- ✅ Vue SPA handles OAuth callback tokens, login/logout state, and surfaces backend health for quick sanity checks.
- ⏳ Still to build: domain aggregates (Movie/Showtime/Seat/Booking) and the user-facing booking/payment APIs that consume seat locks.
- 🔜 Still pending: WebSocket/SSE seat map broadcasting, audit-log ingestion, admin dashboard, notification worker, integration tests.

## Delivery Roadmap & Checkpoints
| # | Checkpoint | Target Outcome | Status |
| - | --- | --- | --- |
| 1 | Docker compose up + frontend hitting `/health` (no OAuth yet) | One-command compose brings up Mongo/Redis/backend/frontend; SPA proxies `/api/health`. | ✅ Completed (compose + health card live).
| 2 | Backend connects to Mongo + Redis with env-driven config | Config loader enforces `MONGO_URI`/`REDIS_ADDR`; API establishes connections on boot. | ✅ Completed (see `internal/config`, `internal/db`, `internal/cache`).
| 3 | Google OAuth 2.0 + JWT + role middleware | OAuth callback issuing JWT with USER/ADMIN roles; middleware guards admin routes. | 🚧 Alpha: happy-path login works but lacks state validation/PKCE + admin UX.
| 4 | Seat lock API with 5-minute Redis TTL + double-lock guard | Endpoints to lock seats, enforce TTL, prevent duplicate holds. | 🚧 Backend implemented; still needs integration tests + frontend UX.
| 5 | WebSocket broadcast for seat status changes | Real-time push (WS/SSE) wired to Redis Pub/Sub for seat map updates. | ⏳ Planned.
| 6 | Booking confirmation (mock payment) → BOOKED + Pub/Sub event | Finalize booking, persist to Mongo, emit success event for notifications. | ⏳ Planned.
| 7 | Timeout handling + seat release + audit logs | Background/job flow to release expired locks, log timeouts/errors, persist audits. | ⏳ Planned.
| 8 | Admin dashboard with filters | Vue admin surface for bookings + filters, audit log stream, role guarding. | ⏳ Planned.

## Running the Stack
### 1. Prerequisites
- Docker Desktop 4.30+ (or compatible engine) with `docker compose` v2.
- Optional: Go 1.24+ and Node 20+ for IDE debugging outside containers.

### 2. Environment Variables
Update the root `.env` file (already committed for local bootstrap) and adjust secrets before running the stack:

| Variable | Description | Example |
| --- | --- | --- |
| `APP_ENV` | Runtime label (`development`, `staging`, `production`) | `development` |
| `PORT` | Backend listen port | `8080` |
| `MONGO_URI` | Mongo connection string (Docker service names supported) | `mongodb://mongo:27017/cinema` |
| `REDIS_ADDR` | Redis host:port | `redis:6379` |
| `JWT_SECRET` | Symmetric signing key for issued JWTs | `change-me-super-secret` |
| `FRONTEND_URL` / `CORS_ORIGINS` | Allowed origins for SPA + admin | `http://localhost:5173` |
| `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET` / `GOOGLE_REDIRECT_URL` | OAuth credentials | `<from Google Console>` |
| `LOG_LEVEL` | `debug`, `info`, etc. | `debug` |

> ⚠️ The committed `.env` exists only for local bootstrap. Replace the secrets (`JWT_SECRET`, Google OAuth keys) with your own values before running anything outside a throwaway sandbox, and keep the real credentials out of source control.

### 3. One-command bootstrap
```
docker compose up --build
```
This brings up Mongo, Redis, the Go API (port 8080), and the Vite dev server (port 5173) with live reload. Health check is available at `http://localhost:8080/health` and the SPA at `http://localhost:5173`.

### 4. Developing outside Docker (optional)
- Backend: `cd backend && go run ./cmd/api` (requires Mongo/Redis running locally or via Docker).
- Frontend: `cd frontend && npm install && npm run dev -- --host 0.0.0.0 --port 5173`.
- Tests (placeholder until suites land): `cd backend && go test ./...`.

## Testing & Verification
- Backend: `cd backend && go test ./...` runs the current suites (seat lock/auth repos still need actual assertions).
- Frontend: `cd frontend && npm run build` ensures the Vite + Vue TypeScript build stays green before shipping Docker images.
- Smoke: `curl http://localhost:8080/health` plus logging into the SPA should be part of every change review until automated tests cover the flow.

## API & Feature Blueprint
### Implemented APIs (alpha)
1. `GET /health` — readiness probe verifying Mongo + Redis connectivity.
2. `POST /api/auth/google/login` — redirects to Google OAuth (state/PKCE hardening pending).
3. `GET /api/auth/google/callback` — exchanges code, upserts users, issues JWT, redirects back to the SPA.
4. `GET /api/me` — returns the authenticated profile and role extracted from JWT claims.
5. `GET /api/admin/ping` — sample ADMIN route guarded by role middleware.
6. `POST /api/showtimes/:id/seats/lock` — acquires Redis-backed locks for seat IDs owned by the caller.
7. `DELETE /api/showtimes/:id/seats/lock` — releases the caller’s seat locks.
8. `GET /api/showtimes/:id/seats/locks` — debug listing of current locks (admin scoping still needed).

### Public/User APIs (planned)
1. `POST /api/auth/google/login` — redirect helper to Google OAuth.
2. `GET /api/auth/google/callback` — exchange code → JWT (role=USER/ADMIN), persist profile.
3. `GET /api/showtimes/:id/seat-map` — cached seat layout with status.
4. `POST /api/bookings` — begin booking; invokes Redis distributed lock for selected seats.
5. `POST /api/bookings/:id/pay` — finalize payment → mark seats BOOKED; emits events.
6. `DELETE /api/bookings/:id` — release lock (user abandon) and emit timeout event.

### Admin APIs (planned)
1. `GET /api/admin/bookings?movie=&date=&user=` — filterable list.
2. `GET /api/admin/audit` — paginated audit log feed.
3. `GET /api/admin/health` — dependency status & build metadata.

### Event & Locking Strategy
- Seat selection triggers `SETNX` (or RedLock with fallback) with 5-minute TTL keyed as `seat:<showtimeId>:<seatId>`.
- Successful locks emit `seat.locked` events on Redis Pub/Sub; release/timeouts emit `seat.released`.
- Booking success persists to Mongo (`bookings` collection) and emits `booking.success` for notification workers.
- Audit service subscribes to Pub/Sub and writes `audit_logs` documents for Success/Timeout/SeatReleased/SystemError states.

## Frontend Responsibilities
- **User surface**: showtime browser, live seat map (WebSocket), booking funnel with payment placeholder, countdown timer for held seats.
- **Admin surface**: role-aware routes, booking table with filters, audit log timeline, real-time health badge.
- **State management**: Pinia (future) + composables for auth/session, WebSocket hooks for seat updates, optimistic UI for seat locking.

## DevOps & Operational Notes
- Backend image is built as a static binary (CGO disabled) and runs as a non-root user on Alpine for smaller footprint & security.
- Frontend dev container uses `su-exec` to avoid permission issues with bind mounts; prod build serves static assets via NGINX.
- Mongo data persisted using the `mongo_data` named volume; remove it manually when you need a clean slate.
- Logging will standardize on structured JSON (`LOG_LEVEL`) with trace IDs to correlate booking + audit events.
- Future CI should run `go test ./...`, `npm run build`, and a docker buildx pipeline to keep images reproducible.

## Compliance Checklist vs Requirements
| Area | Status | Notes |
| --- | --- | --- |
| Authentication (Google OAuth 2.0) | 🚧 Alpha | Login/callback + JWT issuance work; add state/PKCE/nonce + refresh/token rotation. |
| Seat Map (Real-time) | Planned | WebSocket endpoint + Redis Pub/Sub wiring pending. |
| Booking Flow + 5-min Lock | ⏳ Partial | Redis seat locks live; booking persistence/payment handling still to-do. |
| Admin Dashboard + Filters | Planned | Frontend scaffolding ready; API + UI components queued. |
| Audit Logs | Planned | To be stored in Mongo + mirrored via Pub/Sub events. |
| Message Queue Usage | Planned | Redis Pub/Sub chosen; worker service to flush notifications/logging. |
| Concurrency / Double Booking Guards | 🚧 Partial | Lua seat locks exist but there is no TTL sweeper or audit hookup yet. |
| Security / Roles | 🚧 Partial | JWT middleware guards `/api/me` + `/api/admin/ping`; need seed scripts + OAuth hardening. |
| DevOps (single-command compose) | ✅ | `docker compose up --build` brings entire stack online. |
| Optional (Postman/Test/Notification) | Planned | Postman + mock notifier to be added alongside first functional endpoint. |

## Next Steps
1. Replace the committed `.env` with an `.env.example`, rotate JWT + Google secrets, and harden OAuth (`state`, PKCE, nonce, `/auth/callback?error` handling).
2. Design and implement Mongo collections + repositories for Movie, Showtime, SeatMap, and Booking aggregates so the lock service has real data to protect.
3. Flesh out booking + payment endpoints that persist bookings, emit audit events, and reuse the seat lock service end-to-end.
4. Deliver the real-time WebSocket/SSE channel, seat-map UI, and admin dashboards atop Redis Pub/Sub + role guarding.
5. Backfill automated tests (seat lock Lua flows, OAuth handlers, repos) plus Postman/happy-path integration suites and CI wiring.

---
Questions or suggestions? Open an issue or ping the team on Slack `#cinema-ticket-booking`.
