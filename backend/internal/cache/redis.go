package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

func ConnectRedis(ctx context.Context, addr string) (*redis.Client, error) {
	otp := &redis.Options{
		Addr: addr,
		// Password: "",
		// DB: 0,
	}

	rdb := redis.NewClient(otp)

	// Ping when ready to use
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, err
	}

	return rdb, nil

}
