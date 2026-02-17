package seatlock

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

func expZKey(showtimeID string) string {
	return fmt.Sprintf("seatlockexp:%s", showtimeID)
}

// member format: "A1|<owner>|<rid>"
func expMember(seatID, owner, rid string) string {
	return seatID + "|" + owner + "|" + rid
}

func parseMember(m string) (seatID, owner, rid string, ok bool) {
	parts := strings.Split(m, "|")
	if len(parts) != 3 {
		return "", "", "", false
	}
	return parts[0], parts[1], parts[2], true
}

func publishSeatEvent(ctx context.Context, rdb *redis.Client, ev SeatEvent) {
	b, err := json.Marshal(ev)
	if err != nil {
		return
	}
	_ = rdb.Publish(ctx, channel(ev.ShowtimeID), b).Err()
}

// StartTimeoutSweeper runs forever until ctx is cancelled.
func StartTimeoutSweeper(ctx context.Context, rdb *redis.Client) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sweepOnce(ctx, rdb)
		}
	}
}

func sweepOnce(ctx context.Context, rdb *redis.Client) {
	var cursor uint64
	for {
		keys, next, err := rdb.Scan(ctx, cursor, "seatlockexp:*", 50).Result()
		if err != nil {
			return
		}

		for _, zk := range keys {
			showtimeID := strings.TrimPrefix(zk, "seatlockexp:")
			if showtimeID == "" || showtimeID == zk {
				continue
			}
			handleZSet(ctx, rdb, showtimeID, zk)
		}

		cursor = next
		if cursor == 0 {
			break
		}
	}
}

func handleZSet(ctx context.Context, rdb *redis.Client, showtimeID, zk string) {
	nowMs := time.Now().UnixMilli()

	// process up to 200 items per tick per showtime
	for i := 0; i < 200; i++ {
		// ZPOPMIN is atomic: prevents duplicate processing across instances
		zs, err := rdb.ZPopMin(ctx, zk, 1).Result()
		if err != nil || len(zs) == 0 {
			return
		}

		m := zs[0].Member.(string)
		score := int64(zs[0].Score)

		// not yet due => put back and stop
		if score > nowMs {
			_ = rdb.ZAdd(ctx, zk, redis.Z{Score: float64(score), Member: m}).Err()
			return
		}

		seatID, owner, rid, ok := parseMember(m)
		if !ok {
			// drop invalid member
			continue
		}

		lockK := key(showtimeID, seatID)
		bookedK := bookedKey(showtimeID, seatID)

		// 1) booked already -> nothing to do
		if ex, _ := rdb.Exists(ctx, bookedK).Result(); ex == 1 {
			continue
		}

		// 2) lock still exists?
		v, err := rdb.Get(ctx, lockK).Result()
		if err == nil {
			expected := owner + ":" + rid
			if v == expected {
				// not actually expired: reschedule using remaining PTTL
				ttl, tErr := rdb.PTTL(ctx, lockK).Result()
				if tErr == nil && ttl > 0 {
					newExpire := time.Now().UnixMilli() + ttl.Milliseconds()
					_ = rdb.ZAdd(ctx, zk, redis.Z{Score: float64(newExpire), Member: m}).Err()
				}
				// if TTL missing/<=0, we just drop it
			}
			// if owner changed -> drop stale entry
			continue
		}

		// 3) lock missing + not booked => timeout event
		publishSeatEvent(ctx, rdb, SeatEvent{
			Type:       "timeout",
			ShowtimeID: showtimeID,
			SeatIDs:    []string{seatID},
			Owner:      owner,
			RequestID:  rid,
			At:         time.Now().Unix(),
		})
	}
}
