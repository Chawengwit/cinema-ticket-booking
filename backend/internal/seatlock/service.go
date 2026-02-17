package seatlock

import (
	"context"
	"fmt"
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

type LockInfo struct {
	SeatID     string `json:"seat_id"`
	Owner      string `json:"owner"`
	RequestID  string `json:"request_id,omitempty"`
	TTLSeconds int64  `json:"ttl_seconds"`
}

// Atomic multi-seat lock (all-or-nothing)
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

	// res = {1,""} or {0,"seatlock:show:seat"}
	arr, ok := res.([]any)
	if !ok || len(arr) < 2 {
		return false, "", fmt.Errorf("unexpected lua result: %T", res)
	}

	okInt, _ := arr[0].(int64)
	if okInt == 1 {
		return true, "", nil
	}

	confKey, _ := arr[1].(string)
	parts := strings.Split(confKey, ":")
	if len(parts) >= 3 {
		return false, parts[len(parts)-1], nil
	}
	return false, confKey, nil
}

// Release only seats owned by owner (safe)
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
	return err
}

// Debug/dev: list current locks for showtime + ttl remaining
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
