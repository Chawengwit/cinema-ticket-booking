# Cinema Ticket Booking Platform

Cinema Ticket Booking is a full-stack reference implementation for a high-concurrency movie ticketing experience. It demonstrates how to combine Go (Gin) + MongoDB + Redis with a Vue 3 frontend, Google OAuth, and Docker-first DevOps practices while satisfying the requirements for distributed locking, auditability, and real-time seat visibility.

## Feb 17, 2026 Review Snapshot
### Implemented Today
- Google OAuth login + callback persist Google profiles in Mongo, mint JWTs, and expose `/api/me` + `/api/admin/ping` so the Vue shell can show who is signed in.
- Seat locking and booking finalization are wired end-to-end: Redis Lua scripts lock seats, `POST /api/showtimes/:id/bookings/confirm` creates Mongo bookings, and `booking-events` + `seat-events` channels broadcast outcomes.
- Background services now run alongside the API: `seatlock.StartTimeoutSweeper` watches `seatlockexp:*` zsets to emit `seat.timeout` events when holds lapse, and `internal/audit.Run` subscribes to the Redis channels to persist audit logs in Mongo.
- A WebSocket endpoint (`/ws/showtimes/:id/seats`) streams seat lock/release/booked/timeout events via Redis Pub/Sub, and the Vue `SeatEventsCard` demonstrates connecting, locking demo seats, and inspecting the feed.
- Docker Compose + Vite proxy still provide one-command local orchestration, and the SPA handles OAuth callbacks plus health-check surfacing for smoke tests.

### High-priority follow-ups
- Secrets (`JWT_SECRET`, `GOOGLE_CLIENT_*`) remain committed in `.env` (see `.env:1-18`); replace with an `.env.example`, rotate everything, and keep real credentials outside the repo.
- `backend/internal/http/handler/auth_google.go` still sends a fixed `"dev-state"` value and never validates the returned `state`/`error` or uses PKCE/nonce, so the OAuth flow is vulnerable to CSRF and code-injection replay attacks.
- `GET /api/showtimes/:id/seats/locks` is a debug endpoint that any authenticated user can hit and currently leaks lock owners/request IDs (`backend/internal/http/handler/seat_lock.go:134-151`); restrict it to admins or drop it before production.
- The WebSocket gateway accepts JWTs via the query string and unconditionally bypasses origin checks (`backend/internal/http/handler/ws_seats.go:24-49`), so access tokens are exposed in logs and cross-origin callers can subscribe; switch to headers + strict `CheckOrigin`.
- When `MarkBooked` fails after Redis marked seats as booked, the code marks the booking as FAILED but never deletes the `seatbooked:*` keys (`backend/internal/http/handler/booking_confirm.go:125-129`), leaving those seats permanently blocked; add a rollback/retry path.
- No automated tests exist yet; start with table-driven tests for the seat lock Lua scripts, booking repository transitions, and the OAuth/seat APIs.

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
│       ├── audit        # Redis subscribers → Mongo audit logs
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
| Messaging | Redis Pub/Sub → Audit worker + future notification worker | Fire-and-forget notifications, audit persistence, async logging |
| Containers | Docker + docker-compose | Single command local deployment + parity with CI |

## Current Implementation Status (Feb 17, 2026)
- ✅ Docker Compose boot plus `/health` verification covers Mongo, Redis, and Gin with a lean multi-stage backend image.
- ✅ Google OAuth login/callback mints JWTs, stores users in Mongo, powers `/api/me`, and guards `/api/admin/ping` via role middleware.
- ✅ Seat locking + release handlers run on Redis Lua scripts with TTLs, request IDs, audit-friendly payloads, and WebSocket fan-out.
- ✅ Booking confirmation API (`POST /api/showtimes/:id/bookings/confirm`) persists PENDING → BOOKED rows in Mongo and emits `booking-events`.
- ✅ Background timeout sweeper + audit worker run beside the API to publish `seat.timeout` events and store seat/booking activity in Mongo.
- ✅ Vue SPA handles OAuth callback tokens, login/logout state, health checks, and includes a SeatEventsCard to demo the WebSocket feed.
- ⏳ Still to build: real seat-map/query APIs, domain aggregates (Movie/Showtime/Seat/Booking inventory), and the pre-payment booking flow.
- 🔜 Still pending: resilient WebSocket auth/origin controls, audit-log ingestion, admin dashboard, notification worker, and integration tests.

## Delivery Roadmap & Checkpoints
| # | Checkpoint | Target Outcome | Status |
| - | --- | --- | --- |
| 1 | Docker compose up + frontend hitting `/health` (no OAuth yet) | One-command compose brings up Mongo/Redis/backend/frontend; SPA proxies `/api/health`. | ✅ Completed (compose + health card live).
| 2 | Backend connects to Mongo + Redis with env-driven config | Config loader enforces `MONGO_URI`/`REDIS_ADDR`; API establishes connections on boot. | ✅ Completed (see `internal/config`, `internal/db`, `internal/cache`).
| 3 | Google OAuth 2.0 + JWT + role middleware | OAuth callback issuing JWT with USER/ADMIN roles; middleware guards admin routes. | 🚧 Alpha: happy-path login works but lacks state validation/PKCE + admin UX.
| 4 | Seat lock API with 5-minute Redis TTL + double-lock guard | Endpoints to lock seats, enforce TTL, prevent duplicate holds. | ✅ Backend/API complete; still need automated tests + UX polish.
| 5 | WebSocket broadcast for seat status changes | Real-time push (WS/SSE) wired to Redis Pub/Sub for seat map updates. | 🚧 Backend streaming + sample Vue card done; production auth/origin hardening + seat UI pending.
| 6 | Booking confirmation (mock payment) → BOOKED + Pub/Sub event | Finalize booking, persist to Mongo, emit success event for notifications. | 🚧 API + repo exist (mock payment); needs error rollback + real payment adapter.
| 7 | Timeout handling + seat release + audit logs | Background/job flow to release expired locks, log timeouts/errors, persist audits. | 🚧 Partial: timeout sweeper + audit worker live; still need admin surfacing + retention/governance.
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
- Backend: `cd backend && go test ./...` runs the current suites (set `GOCACHE=$(pwd)/.gocache` if your environment blocks `~/Library/Caches`). Real assertions for seat locks, bookings, and auth handlers are still missing.
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
9. `POST /api/showtimes/:id/bookings/confirm` — mock payment + booking persistence that atomically flips locks to booked seats.
10. `GET /ws/showtimes/:id/seats?token=JWT` — WebSocket streaming seat lock/release/book/timeout events from Redis Pub/Sub (needs origin/token hardening).

### Public/User APIs (planned)
1. `GET /api/showtimes/:id/seat-map` — cached seat layout with AVAILABLE/LOCKED/BOOKED status.
2. `POST /api/bookings` — begin a booking/hold flow (idempotent) separate from the confirm endpoint.
3. `POST /api/bookings/:id/pay` — finalize booking against a payment provider and hand off to `ConfirmSeatsBooked`.
4. `DELETE /api/bookings/:id` — release lock (user abandon) and emit timeout event.
5. `GET /api/bookings/me` — list user bookings for the SPA/account center.

### Admin APIs (planned)
1. `GET /api/admin/bookings?movie=&date=&user=` — filterable list.
2. `GET /api/admin/audit` — paginated audit log feed.
3. `GET /api/admin/health` — dependency status & build metadata.

### Event & Locking Strategy
- Seat selection triggers a Lua-powered `SET` with PX TTL keyed as `seatlock:<showtimeId>:<seatId>` and records an expiry marker in `seatlockexp:<showtimeId>` with payload `seat|owner|requestId` so locks are atomic, deduped, and sweepable.
- `seatlock.StartTimeoutSweeper` scans the `seatlockexp:*` sorted sets, pops due members, checks whether the seat is still locked/booked, and emits `seat.timeout` events when a hold lapses.
- Booking confirmation runs another Lua script that ensures no `seatbooked:*` keys exist yet, validates ownership (`owner:requestID`), creates `seatbooked:*` markers, and deletes the original locks plus expiry markers.
- Every lock/release/booked/timeout transition emits a JSON payload on `seat-events:<showtimeId>`; `/ws/showtimes/:id/seats` streams that channel to connected browsers.
- Booking success also emits `booking.success` on the `booking-events` channel for notification workers / audit logging.

-## Background Workers
- **Timeout sweeper** (`internal/seatlock/timeout_sweeper.go`): runs every second, scans `seatlockexp:*`, republishes due members if their TTL was refreshed, or emits `seat.timeout` events when a hold disappears without being booked.
- **Audit worker** (`internal/audit/worker.go`): `PSubscribe`s to `seat-events:*` + `booking-events`, normalizes them into `seat.*` / `booking.*` audit documents, and inserts them into Mongo via `repo.AuditRepo`.
- Both workers currently run in-process (see `cmd/api/main.go`) and inherit the API’s lifecycle; plan for graceful shutdown, observability, and horizontal scale when splitting into dedicated services.

## Frontend Responsibilities
- **User surface**: showtime browser, live seat map (WebSocket), booking funnel with payment placeholder, countdown timer for held seats — today only the SeatEventsCard dev tool exists, so the real UX is still to build.
- **Admin surface**: role-aware routes, booking table with filters, audit log timeline, real-time health badge.
- **State management**: Pinia (future) + composables for auth/session, WebSocket hooks for seat updates, optimistic UI for seat locking.

## DevOps & Operational Notes
- Backend image is built as a static binary (CGO disabled) and runs as a non-root user on Alpine for smaller footprint & security.
- Frontend dev container uses `su-exec` to avoid permission issues with bind mounts; prod build serves static assets via NGINX.
- Mongo data persisted using the `mongo_data` named volume; remove it manually when you need a clean slate.
- Logging will standardize on structured JSON (`LOG_LEVEL`) with trace IDs to correlate booking + audit events.
- Future CI should run `go test ./...`, `npm run build`, and a docker buildx pipeline to keep images reproducible.
- The API process double-duties background jobs (timeout sweeper + audit worker). Plan for graceful shutdown hooks and eventually separating them into their own deployments if they grow heavier.

## Compliance Checklist vs Requirements
| Area | Status | Notes |
| --- | --- | --- |
| Authentication (Google OAuth 2.0) | 🚧 Alpha | Login/callback + JWT issuance work; add state/PKCE/nonce + refresh/token rotation. |
| Seat Map (Real-time) | 🚧 Partial | Redis Pub/Sub + WebSocket streaming in place; need seat-map query API + real UI. |
| Booking Flow + 5-min Lock | 🚧 Partial | Lock + confirm API exists; still need booking creation/cancel endpoints and rollback on DB failure. |
| Admin Dashboard + Filters | Planned | Frontend scaffolding ready; API + UI components queued. |
| Audit Logs | 🚧 Partial | Seat + booking events persist to Mongo via the audit worker; need APIs/retention/alerting. |
| Message Queue Usage | 🚧 Partial | `seat-events:*` + `booking-events` exist, audit worker consumes them; notification worker still TBD. |
| Concurrency / Double Booking Guards | 🚧 Partial | Lua seat locks/booked markers exist but there is no TTL sweeper or audit hookup yet. |
| Security / Roles | 🚧 Partial | JWT middleware guards REST APIs, but OAuth state/PKCE, debug lock list, and WebSocket token/origin handling remain to-do. |
| DevOps (single-command compose) | ✅ | `docker compose up --build` brings entire stack online. |
| Optional (Postman/Test/Notification) | Planned | Postman + mock notifier to be added alongside first functional endpoint. |

## Next Steps
1. Replace the committed `.env` with an `.env.example`, rotate all secrets, and harden auth: random OAuth `state` + PKCE + nonce, reject `/auth/callback?error`, move WebSocket auth to headers, and enforce `CheckOrigin`.
2. Lock down operational endpoints (e.g., `/api/showtimes/:id/seats/locks`) so only admins or internal tooling can view lock owners/request IDs; expose the same data via structured audit logs instead.
3. Design and implement Mongo collections + repositories for Movie, Showtime, SeatMap, and booking lifecycle APIs (create/cancel/list/seat-map) so the lock/booking/audit flows operate on real inventory.
4. Fix the booking rollback story: if Mongo updates fail after Redis marks seats as booked, delete/retry affected `seatbooked:*` keys, emit compensating events, and wire timeout sweeps into audit + alerting.
5. Build the real seat-map + booking UI, admin dashboards, audit log surfaces, and add automated tests (seat lock Lua flows, booking repo transitions, OAuth handlers) plus Postman/happy-path integration suites and CI wiring.

---
Questions or suggestions? Open an issue or ping the team on Slack `#cinema-ticket-booking`.
