package main

import (
	"cinema/internal/cache"
	"cinema/internal/config"
	"cinema/internal/db"
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	// load config
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	// connect mongo + redis
	rootCtx := context.Background()

	mongoConn, err := db.ConnectMongo(rootCtx, cfg.MongoURI)
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = mongoConn.Client.Disconnect(context.Background())
	}()

	redisClient, err := cache.ConnectRedis(rootCtx, cfg.RedisAddr)
	if err != nil {
		panic(err)
	}

	defer func() {
		_ = redisClient.Close()
	}()

	// router
	r := gin.Default()
	_ = r.SetTrustedProxies(nil)

	// health
	r.GET("/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		mongoOK := mongoConn.Client.Ping(ctx, nil) == nil
		redisOK := redisClient.Ping(ctx).Err() == nil

		c.JSON(http.StatusOK, gin.H{
			"ok":       true,
			"env":      cfg.AppEnv,
			"port":     cfg.Port,
			"mongo_ok": mongoOK,
			"redis_ok": redisOK,
		})
	})

	_ = r.Run(":" + cfg.Port)
}
