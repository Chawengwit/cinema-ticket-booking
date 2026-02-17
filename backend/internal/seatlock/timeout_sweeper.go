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
	// scan all showtime exp sets
	var cursor uint64
	for {
		keys, next, err := rdb.Scan(ctx, cursor, "seatlockexp:*", 50).Result()
		if err != nil {
			return
		}
		for _, zk := range keys {
			// zk = seatlockexp:<showtimeId>
			parts := strings.Split(zk, ":")
			if len(parts) < 2 {
				continue
			}
			showtimeID := parts[1]
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

	// pull up to 200 expired items per tick per showtime (กัน loop ยาว)
	members, err := rdb.ZRangeByScore(ctx, zk, &redis.ZRangeBy{
		Min:    "-inf",
		Max:    fmt.Sprintf("%d", nowMs),
		Offset: 0,
		Count:  200,
	}).Result()
	if err != nil || len(members) == 0 {
		return
	}

	for _, m := range members {
		seatID, owner, rid, ok := parseMember(m)
		if !ok {
			_ = rdb.ZRem(ctx, zk, m).Err()
			continue
		}

		lockK := key(showtimeID, seatID)
		bookedK := bookedKey(showtimeID, seatID)

		// 1) ถ้า booked แล้ว => ลบทิ้งจาก zset
		if ex, _ := rdb.Exists(ctx, bookedK).Result(); ex == 1 {
			_ = rdb.ZRem(ctx, zk, m).Err()
			continue
		}

		// 2) ถ้า lock ยังอยู่ ให้เช็ค owner:rid
		v, err := rdb.Get(ctx, lockK).Result()
		if err == nil {
			expected := owner + ":" + rid
			if v == expected {
				// ยังไม่หมดจริง: refresh score จาก TTL ที่เหลือ
				ttl, tErr := rdb.PTTL(ctx, lockK).Result()
				if tErr == nil && ttl > 0 {
					newExpire := time.Now().Add(ttl).UnixMilli()
					_ = rdb.ZAdd(ctx, zk, redis.Z{Score: float64(newExpire), Member: m}).Err()
				} else {
					_ = rdb.ZRem(ctx, zk, m).Err()
				}
			} else {
				// lock ถูกเปลี่ยนเจ้าของแล้ว → entry นี้ stale
				_ = rdb.ZRem(ctx, zk, m).Err()
			}
			continue
		}

		// 3) lock หาย + booked ไม่มา => timeout จริง
		_ = rdb.ZRem(ctx, zk, m).Err()

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
