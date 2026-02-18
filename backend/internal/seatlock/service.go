package seatlock

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type Service struct {
	rdb *redis.Client
	ttl time.Duration
}

func New(rdb *redis.Client, ttl time.Duration) *Service {
	return &Service{rdb: rdb, ttl: ttl}
}

func key(showtimeID, seatID string) string {
	return fmt.Sprintf("seatlock:%s:%s", showtimeID, seatID)
}

func bookedKey(showtimeID, seatID string) string {
	return fmt.Sprintf("seatbooked:%s:%s", showtimeID, seatID)
}

type LockInfo struct {
	SeatID     string `json:"seat_id"`
	Owner      string `json:"owner"`
	RequestID  string `json:"request_id,omitempty"`
	TTLSeconds int64  `json:"ttl_seconds"`
}

// =====================
// Seat events (PubSub)
// =====================

type SeatEvent struct {
	Type       string   `json:"type"` // "locked" | "released" | "booked" | "timeout"
	ShowtimeID string   `json:"showtime_id"`
	SeatIDs    []string `json:"seat_ids"`
	Owner      string   `json:"owner"`
	RequestID  string   `json:"request_id,omitempty"`
	BookingID  string   `json:"booking_id,omitempty"`
	At         int64    `json:"at"` // unix seconds
}

func channel(showtimeID string) string {
	return fmt.Sprintf("seat-events:%s", showtimeID)
}

func (s *Service) publish(ctx context.Context, ev SeatEvent) {
	b, err := json.Marshal(ev)
	if err != nil {
		return
	}
	_ = s.rdb.Publish(ctx, channel(ev.ShowtimeID), b).Err()
}

// =====================
// Atomic lock all seats
// =====================

// value stored as: owner:requestId
// - allow lock if key empty OR already owned by same owner (prefix match)
var luaLockAll = redis.NewScript(`
local owner = ARGV[1]
local value = ARGV[2]
local ttlMs = tonumber(ARGV[3])

local function starts_with(str, prefix)
  return string.sub(str, 1, string.len(prefix)) == prefix
end

-- check conflicts first
for i=1,#KEYS do
  local v = redis.call("GET", KEYS[i])
  if v and (not starts_with(v, owner .. ":")) then
    return {0, KEYS[i]}
  end
end

-- lock all
for i=1,#KEYS do
  redis.call("SET", KEYS[i], value, "PX", ttlMs)
end

return {1, ""}
`)

func (s *Service) LockSeats(ctx context.Context, showtimeID string, seatIDs []string, owner string, requestID string) (locked bool, conflictedSeatID string, err error) {
	if len(seatIDs) == 0 {
		return false, "", fmt.Errorf("seatIDs required")
	}
	if owner == "" {
		return false, "", fmt.Errorf("owner required")
	}
	if requestID == "" {
		return false, "", fmt.Errorf("requestID required")
	}

	keys := make([]string, 0, len(seatIDs))
	for _, sid := range seatIDs {
		keys = append(keys, key(showtimeID, sid))
	}

	value := owner + ":" + requestID

	res, err := luaLockAll.Run(ctx, s.rdb, keys, owner, value, s.ttl.Milliseconds()).Result()
	if err != nil {
		return false, "", err
	}

	arr, ok := res.([]any)
	if !ok || len(arr) < 2 {
		return false, "", fmt.Errorf("unexpected lua result: %T", res)
	}

	okInt, _ := arr[0].(int64)
	if okInt == 1 {
		// track expiry for timeout sweeper
		expireMs := time.Now().Add(s.ttl).UnixMilli()
		zk := expZKey(showtimeID)

		pipe := s.rdb.Pipeline()
		for _, sid := range seatIDs {
			pipe.ZAdd(ctx, zk, redis.Z{
				Score:  float64(expireMs),
				Member: expMember(sid, owner, requestID),
			})
		}
		_, _ = pipe.Exec(ctx)

		s.publish(ctx, SeatEvent{
			Type:       "locked",
			ShowtimeID: showtimeID,
			SeatIDs:    seatIDs,
			Owner:      owner,
			RequestID:  requestID,
			At:         time.Now().Unix(),
		})
		return true, "", nil
	}

	confKey, _ := arr[1].(string)
	parts := strings.Split(confKey, ":")
	if len(parts) >= 3 {
		return false, parts[len(parts)-1], nil
	}
	return false, confKey, nil
}

// =====================
// Release seats owned by owner
// =====================

// matches prefix: owner:
var luaReleaseOwned = redis.NewScript(`
local owner = ARGV[1]

local function starts_with(str, prefix)
  return string.sub(str, 1, string.len(prefix)) == prefix
end

for i=1,#KEYS do
  local v = redis.call("GET", KEYS[i])
  if v and starts_with(v, owner .. ":") then
    redis.call("DEL", KEYS[i])
  end
end
return 1
`)

// remove ZSET members that start with "seat|owner|"
var luaZRemBySeatOwner = redis.NewScript(`
local zkey = KEYS[1]
local owner = ARGV[1]

local members = redis.call("ZRANGE", zkey, 0, -1)

for i=2,#KEYS do
  local seatId = KEYS[i]
  local prefix = seatId .. "|" .. owner .. "|"
  for _,m in ipairs(members) do
    if string.sub(m, 1, string.len(prefix)) == prefix then
      redis.call("ZREM", zkey, m)
    end
  end
end

return 1
`)

func (s *Service) ReleaseSeats(ctx context.Context, showtimeID string, seatIDs []string, owner string) error {
	if len(seatIDs) == 0 {
		return nil
	}
	if owner == "" {
		return fmt.Errorf("owner required")
	}

	keys := make([]string, 0, len(seatIDs))
	for _, sid := range seatIDs {
		keys = append(keys, key(showtimeID, sid))
	}

	_, err := luaReleaseOwned.Run(ctx, s.rdb, keys, owner).Result()
	if err != nil {
		return err
	}

	// cleanup expiry tracking (any rid of this owner)
	zk := expZKey(showtimeID)
	remKeys := make([]string, 0, len(seatIDs)+1)
	remKeys = append(remKeys, zk)
	for _, sid := range seatIDs {
		remKeys = append(remKeys, sid)
	}
	_, _ = luaZRemBySeatOwner.Run(ctx, s.rdb, remKeys, owner).Result()

	s.publish(ctx, SeatEvent{
		Type:       "released",
		ShowtimeID: showtimeID,
		SeatIDs:    seatIDs,
		Owner:      owner,
		At:         time.Now().Unix(),
	})
	return nil
}

// =====================
// Confirm booking atomically
// =====================

// KEYS layout: [1..n] lock keys, [n+1..2n] booked keys
var luaConfirmBooked = redis.NewScript(`
local owner = ARGV[1]
local rid = ARGV[2]
local bookingId = ARGV[3]
local n = tonumber(ARGV[4])

local expected = owner .. ":" .. rid

-- First: if any seat already booked
for i=1,n do
  local bookedK = KEYS[n+i]
  if redis.call("EXISTS", bookedK) == 1 then
    return {0, bookedK, "already_booked"}
  end
end

-- Second: validate locks exist and owned by same owner+rid
for i=1,n do
  local lockK = KEYS[i]
  local v = redis.call("GET", lockK)
  if (not v) then
    return {0, lockK, "missing_lock"}
  end
  if v ~= expected then
    return {0, lockK, "not_owner"}
  end
end

-- Finalize: mark booked + delete locks
for i=1,n do
  local lockK = KEYS[i]
  local bookedK = KEYS[n+i]
  redis.call("SET", bookedK, bookingId)
  redis.call("DEL", lockK)
end

return {1, "", ""}
`)

func (s *Service) ConfirmSeatsBooked(
	ctx context.Context,
	showtimeID string,
	seatIDs []string,
	owner string,
	requestID string,
	bookingID string,
) (ok bool, conflictedSeatID string, reason string, err error) {
	if len(seatIDs) == 0 {
		return false, "", "invalid_seat_ids", fmt.Errorf("seatIDs required")
	}
	if owner == "" || requestID == "" || bookingID == "" {
		return false, "", "invalid_args", fmt.Errorf("owner/requestID/bookingID required")
	}

	keys := make([]string, 0, len(seatIDs)*2)
	for _, sid := range seatIDs {
		keys = append(keys, key(showtimeID, sid))
	}
	for _, sid := range seatIDs {
		keys = append(keys, bookedKey(showtimeID, sid))
	}

	res, err := luaConfirmBooked.Run(ctx, s.rdb, keys, owner, requestID, bookingID, len(seatIDs)).Result()
	if err != nil {
		return false, "", "redis_failed", err
	}

	arr, okArr := res.([]any)
	if !okArr || len(arr) < 3 {
		return false, "", "unexpected_lua_result", fmt.Errorf("unexpected lua result: %T", res)
	}

	okInt, _ := arr[0].(int64)

	zk := expZKey(showtimeID)

	// ✅ SUCCESS: cleanup expiry tracking + publish
	if okInt == 1 {
		pipe := s.rdb.Pipeline()
		for _, sid := range seatIDs {
			pipe.ZRem(ctx, zk, expMember(sid, owner, requestID))
		}
		_, _ = pipe.Exec(ctx)

		s.publish(ctx, SeatEvent{
			Type:       "booked",
			ShowtimeID: showtimeID,
			SeatIDs:    seatIDs,
			Owner:      owner,
			RequestID:  requestID,
			BookingID:  bookingID,
			At:         time.Now().Unix(),
		})

		return true, "", "", nil
	}

	// failure
	confKey, _ := arr[1].(string)
	reason, _ = arr[2].(string)

	// ✅ IMPORTANT:
	// - If already_booked -> remove zset member to avoid sweeper firing timeout later (stale)
	// - If missing_lock -> do NOT remove (let sweeper produce timeout event)
	// - If not_owner -> do NOT remove (owner mismatch)
	if reason == "already_booked" {
		pipe := s.rdb.Pipeline()
		for _, sid := range seatIDs {
			pipe.ZRem(ctx, zk, expMember(sid, owner, requestID))
		}
		_, _ = pipe.Exec(ctx)
	}

	// extract seat id from key (seatlock/show/seat or seatbooked/show/seat)
	if strings.HasPrefix(confKey, "seatlock:") || strings.HasPrefix(confKey, "seatbooked:") {
		parts := strings.Split(confKey, ":")
		if len(parts) >= 3 {
			return false, parts[len(parts)-1], reason, nil
		}
	}

	return false, confKey, reason, nil
}

// =====================
// Debug/dev: list current locks for showtime + ttl remaining
// =====================

func (s *Service) ListLocks(ctx context.Context, showtimeID string) ([]LockInfo, error) {
	pattern := fmt.Sprintf("seatlock:%s:*", showtimeID)

	var cursor uint64
	out := make([]LockInfo, 0, 32)

	for {
		keys, next, err := s.rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, err
		}

		if len(keys) > 0 {
			pipe := s.rdb.Pipeline()
			valCmds := make([]*redis.StringCmd, len(keys))
			ttlCmds := make([]*redis.DurationCmd, len(keys))

			for i, k := range keys {
				valCmds[i] = pipe.Get(ctx, k)
				ttlCmds[i] = pipe.TTL(ctx, k)
			}

			_, _ = pipe.Exec(ctx)

			for i, k := range keys {
				v, vErr := valCmds[i].Result()
				ttl, tErr := ttlCmds[i].Result()
				if vErr != nil || tErr != nil {
					continue
				}

				parts := strings.Split(k, ":")
				seatID := k
				if len(parts) >= 3 {
					seatID = parts[len(parts)-1]
				}

				owner := v
				rid := ""
				if vv := strings.SplitN(v, ":", 2); len(vv) == 2 {
					owner = vv[0]
					rid = vv[1]
				}

				sec := int64(ttl.Seconds())
				if sec < 0 {
					sec = 0
				}

				out = append(out, LockInfo{
					SeatID:     seatID,
					Owner:      owner,
					RequestID:  rid,
					TTLSeconds: sec,
				})
			}
		}

		cursor = next
		if cursor == 0 {
			break
		}
	}

	return out, nil
}

// Debug/dev: list booked seats for showtime
func (s *Service) ListBookedSeats(ctx context.Context, showtimeID string) ([]string, error) {
	pattern := fmt.Sprintf("seatbooked:%s:*", showtimeID)

	var cursor uint64
	out := make([]string, 0, 64)

	for {
		keys, next, err := s.rdb.Scan(ctx, cursor, pattern, 200).Result()
		if err != nil {
			return nil, err
		}

		for _, k := range keys {
			parts := strings.Split(k, ":")
			if len(parts) >= 3 {
				out = append(out, parts[len(parts)-1])
			} else {
				out = append(out, k)
			}
		}

		cursor = next
		if cursor == 0 {
			break
		}
	}

	sort.Strings(out)
	return out, nil
}
