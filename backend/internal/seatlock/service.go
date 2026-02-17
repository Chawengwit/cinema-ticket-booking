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

// Atomic multi-seat lock (all-or-nothing)
// - allow lock if key empty OR already owned by same owner (re-entrant)
var luaLockAll = redis.NewScript(`
local owner = ARGV[1]
local ttlMs = tonumber(ARGV[2])

-- check conflicts first
for i=1,#KEYS do
  local v = redis.call("GET", KEYS[i])
  if v and v ~= owner then
    return {0, KEYS[i]}
  end
end

-- lock all
for i=1,#KEYS do
  redis.call("SET", KEYS[i], owner, "PX", ttlMs)
end

return {1, ""}
`)

func (s *Service) LockSeats(ctx context.Context, showtimeID string, seatIDs []string, owner string) (locked bool, conflictedSeatID string, err error) {
	if len(seatIDs) == 0 {
		return false, "", fmt.Errorf("seatIDs required")
	}

	keys := make([]string, 0, len(seatIDs))
	for _, sid := range seatIDs {
		keys = append(keys, key(showtimeID, sid))
	}

	res, err := luaLockAll.Run(ctx, s.rdb, keys, owner, s.ttl.Milliseconds()).Result()
	if err != nil {
		return false, "", err
	}

	arr, ok := res.([]any)
	if !ok || len(arr) < 2 {
		return false, "", fmt.Errorf("unexpected lua result: %T", res)
	}

	okInt, _ := arr[0].(int64)
	if okInt == 1 {
		return true, "", nil
	}

	confKey, _ := arr[1].(string)
	// seatlock:showtime:seat -> extract seat
	parts := strings.Split(confKey, ":")
	if len(parts) >= 3 {
		return false, parts[len(parts)-1], nil
	}
	return false, confKey, nil
}

// Release only seats owned by owner (safe)
var luaReleaseOwned = redis.NewScript(`
local owner = ARGV[1]
for i=1,#KEYS do
  local v = redis.call("GET", KEYS[i])
  if v and v == owner then
    redis.call("DEL", KEYS[i])
  end
end
return 1
`)

func (s *Service) ReleaseSeats(ctx context.Context, showtimeID string, seatIDs []string, owner string) error {
	if len(seatIDs) == 0 {
		return nil
	}

	keys := make([]string, 0, len(seatIDs))
	for _, sid := range seatIDs {
		keys = append(keys, key(showtimeID, sid))
	}

	_, err := luaReleaseOwned.Run(ctx, s.rdb, keys, owner).Result()
	return err
}
