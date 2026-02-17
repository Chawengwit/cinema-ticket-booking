# Cinema Ticket Booking Platform

Cinema Ticket Booking is a full-stack reference implementation for a high-concurrency movie ticketing experience. It demonstrates how to combine Go (Gin) + MongoDB + Redis with a Vue 3 frontend, Google OAuth, and Docker-first DevOps practices while satisfying the requirements for distributed locking, auditability, and real-time seat visibility.

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
│       ├── config       # Env configuration
│       ├── domain       # Core business entities (Movie, Showtime, Booking)
│       ├── repository   # MongoDB implementations
│       ├── db           # Mongo connection setup
│       └── cache        # Redis connection setup
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

## Current Implementation Status (Feb 16, 2026)
- ✅ Bootstrapped health-checked Go API (`/health`) with Mongo + Redis clients wired and lint-friendly Docker image.
- ✅ Vue shell rendering backend health via Vite dev server (proxy through docker compose).
- ⏳ In progress: domain models (Movie, Showtime, Seat, Booking), REST APIs, WebSocket gateway, Google OAuth callback, Redis lock helpers, audit log writer.
- 🔜 Upcoming: Admin dashboard views, notification worker, Postman collection, happy-path integration tests.

## Delivery Roadmap & Checkpoints
| # | Checkpoint | Target Outcome | Status |
| - | --- | --- | --- |
| 1 | Docker compose up + frontend hitting `/health` (no OAuth yet) | One-command compose brings up Mongo/Redis/backend/frontend; SPA proxies `/api/health`. | ✅ Completed (compose + health card live).
| 2 | Backend connects to Mongo + Redis with env-driven config | Config loader enforces `MONGO_URI`/`REDIS_ADDR`; API establishes connections on boot. | ✅ Completed (see `internal/config`, `internal/db`, `internal/cache`).
| 3 | Google OAuth 2.0 + JWT + role middleware | OAuth callback issuing JWT with USER/ADMIN roles; middleware guards admin routes. | ⏳ Planned.
| 4 | Seat lock API with 5-minute Redis TTL + double-lock guard | Endpoints to lock seats, enforce TTL, prevent duplicate holds. | ⏳ Planned.
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

### 3. One-command bootstrap
```
docker compose up --build
```
This brings up Mongo, Redis, the Go API (port 8080), and the Vite dev server (port 5173) with live reload. Health check is available at `http://localhost:8080/health` and the SPA at `http://localhost:5173`.

### 4. Developing outside Docker (optional)
- Backend: `cd backend && go run ./cmd/api` (requires Mongo/Redis running locally or via Docker).
- Frontend: `cd frontend && npm install && npm run dev -- --host 0.0.0.0 --port 5173`.
- Tests (placeholder until suites land): `cd backend && go test ./...`.

## API & Feature Blueprint
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
| Authentication (Google OAuth 2.0) | Planned | Env placeholders exist; handler + token service to be implemented next sprint. |
| Seat Map (Real-time) | Planned | WebSocket endpoint + Redis Pub/Sub wiring pending. |
| Booking Flow + 5-min Lock | Planned | Redis client + env ready; need seat + booking repos and lock helpers. |
| Admin Dashboard + Filters | Planned | Frontend scaffolding ready; API + UI components queued. |
| Audit Logs | Planned | To be stored in Mongo + mirrored via Pub/Sub events. |
| Message Queue Usage | Planned | Redis Pub/Sub chosen; worker service to flush notifications/logging. |
| Concurrency / Double Booking Guards | Designing | Seat-level locks, idempotency keys, and TTL sweeper described above. |
| Security / Roles | Designing | OAuth callback will mint JWT with `role` claim; admin routes protected via middleware. |
| DevOps (single-command compose) | ✅ | `docker compose up --build` brings entire stack online. |
| Optional (Postman/Test/Notification) | Partially planned | Postman + mock notifier to be added alongside first functional endpoint. |

## Next Steps
1. **[IN PROGRESS]** Implement Mongo repositories + domain aggregates (See `docs/TASK_01_DOMAIN_MODELS.md`).
2. Build OAuth controllers and session middleware; add JWT signer/validator.
3. Deliver seat map WebSocket channel + Redis Pub/Sub broadcaster.
4. Flesh out booking + payment endpoints with lock lifecycle + audit hooks.
5. Stand up admin dashboard pages plus Postman collection + happy-path integration tests.

---
Questions or suggestions? Open an issue or ping the team on Slack `#cinema-ticket-booking`.
