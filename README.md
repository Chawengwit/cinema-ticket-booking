# Cinema Ticket Booking Platform

Clean, production‑oriented overview of the Go + Vue full-stack system for real‑time movie seat booking with Redis-backed locking and Mongo persistence.

## 1) System Architecture Diagram
```
[Vue 3 SPA] --HTTP/WS--> [Gin API]
                            |
                  +---------+---------+
                  |                   |
             [MongoDB]           [Redis]
                 |           (locks + pub/sub)
                 |                   |
            [audit_logs]     seat-events / booking-events
                  |
         [Audit worker (in-process)]
```

## 2) Tech Stack Overview
- Backend: Go 1.24, Gin, MongoDB, Redis, JWT auth, Docker.
- Frontend: Vue 3 + Vite + Tailwind CSS.
- Auth: Google OAuth 2.0, JWT bearer tokens; admin role via `ADMIN_EMAILS`.
- Realtime: Redis Pub/Sub → WebSocket endpoint `/ws/showtimes/:id/seats`.
- Container orchestration: Docker Compose (services: mongo, redis, backend, frontend).

## 3) Booking Flow (step-by-step)
1) User clicks “Sign in with Google” → backend `/api/auth/google/login` → Google → callback `/api/auth/google/callback` issues JWT + stores/updates user in Mongo.  
2) SPA stores JWT and calls `/api/me` to show profile + role.  
3) User selects showtime and seats; client posts to `/api/showtimes/:showtimeId/seats/lock` with `seat_ids`.  
4) Backend runs Lua-based Redis locks (5‑minute TTL) to ensure all-or-nothing holds; emits `seat.locked` on `seat-events:<showtimeId>`.  
5) Payment (mock) + confirmation: client calls `/api/showtimes/:showtimeId/bookings/confirm` with `request_id` + seats; service atomically flips locks to booked keys, writes booking to Mongo, and emits `booking.success`.  
6) Locks are released either by explicit DELETE `/api/showtimes/:showtimeId/seats/lock` or by timeout sweeper emitting `seat.timeout`.  
7) WebSocket subscribers stream seat events for live UI updates.

## 4) Redis Lock Strategy
- Keys: `seatlock:<showtimeId>:<seatId>` (value `owner:requestId`), TTL configurable via `SEAT_LOCK_TTL_SECONDS` (default 300s).  
- Booking markers: `seatbooked:<showtimeId>:<seatId>` set on successful confirmation.  
- Expiry tracking: sorted set `seatlockexp:<showtimeId>` members `seat|owner|requestId` to drive timeout sweeper.  
- Ownership rules: lock allowed only if empty or already owned by same `owner` prefix; release only by owner.  
- Lua scripts:  
  - Lock all seats atomically (returns conflicted seat).  
  - Release owned seats.  
  - Confirm booking (validate ownership + not booked, then set booked keys and delete locks).  
- Timeout: in-process sweeper pops due entries; if lock missing and not booked, publishes `seat.timeout`.  
- Idempotency: `request_id` travels through lock + booking confirm so retries stay consistent.

## 5) Message Queue (Redis Pub/Sub)
- Channels:  
  - `seat-events:<showtimeId>` — published by seat lock service for `locked`, `released`, `booked`, `timeout`.  
  - `booking-events` — published on booking success.  
- Consumers:  
  - WebSocket endpoint `/ws/showtimes/:showtimeId/seats` streams `seat-events`.  
  - Audit worker subscribes to both channels and writes `audit_logs` in Mongo.  
- Rationale: lightweight, in-memory fan-out for real-time UX and auditing; upgrade path to a durable queue if needed.

## 6) How to Run
**Prerequisites**: Docker (compose v2), optionally Go 1.24 and Node 20 for local dev.  
**Env file**: create `.env` in repo root (example values below). **Do not commit real secrets.**
```
APP_ENV=development
PORT=8080
MONGO_URI=mongodb://mongo:27017/cinema
REDIS_ADDR=redis:6379
JWT_SECRET=change-me-32chars-min
FRONTEND_URL=http://localhost:5173
CORS_ORIGINS=http://localhost:5173
GOOGLE_CLIENT_ID=your-google-client-id
GOOGLE_CLIENT_SECRET=your-google-client-secret
GOOGLE_REDIRECT_URL=http://localhost:8080/api/auth/google/callback
LOG_LEVEL=debug
SEAT_LOCK_TTL_SECONDS=300
ADMIN_EMAILS=admin@example.com
```
**Compose up (recommended)**  
```bash
docker compose up --build
```
Services: backend `:8080`, frontend `:5173`, Mongo `:27017`, Redis `:6379`.  
**Smoke checks**:  
- `curl http://localhost:8080/health` → mongo_ok/redis_ok true.  
- Open `http://localhost:5173/`, log in with Google, use Seat Events card to connect WS and lock seats.

## 7) Assumptions & Trade-offs
- Redis Pub/Sub chosen for simplicity; not durable—would swap for Kafka/NATS/Rabbit for guaranteed delivery.  
- Timeout sweeper and audit worker run in-process with the API for ease of deployment; could be split into separate services for resilience.  
- Admin role controlled by `ADMIN_EMAILS` allowlist; no UI for role management yet—rotate carefully.  
- Payment is mocked; replace with real PSP and idempotent payment intents before production.  
- WebSocket auth uses JWT query param today; plan to move to headers/cookies and stricter origin checks.  
- No integration tests yet; rely on manual smoke plus unit coverage to be added.
