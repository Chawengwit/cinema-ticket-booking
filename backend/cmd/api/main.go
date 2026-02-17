package main

import (
	"cinema/internal/auth"
	"cinema/internal/cache"
	"cinema/internal/config"
	"cinema/internal/db"
	"cinema/internal/http/handler"
	"cinema/internal/http/middleware"
	"cinema/internal/model"
	"cinema/internal/repo"
	"cinema/internal/seatlock"
	"context"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
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
	defer func() { _ = mongoConn.Client.Disconnect(context.Background()) }()

	redisClient, err := cache.ConnectRedis(rootCtx, cfg.RedisAddr)
	if err != nil {
		panic(err)
	}
	defer func() { _ = redisClient.Close() }()

	// services
	jwtSvc := auth.NewJWTService(cfg.JWTSecret)
	userRepo := repo.NewUserRepo(mongoConn.DB)

	// WebSocket
	seatWS := handler.NewSeatWSHandler(redisClient, jwtSvc)

	// SeatLock service
	seatTTL := time.Duration(cfg.SeatLockTTLSeconds) * time.Second
	seatLockSvc := seatlock.New(redisClient, seatTTL)
	seatLockHandler := handler.NewSeatLockHandler(seatLockSvc, cfg.SeatLockTTLSeconds)

	// Booking
	bookingRepo := repo.NewBookingRepo(mongoConn.DB)
	bookingHandler := handler.NewBookingHandler(seatLockSvc, bookingRepo, redisClient)

	// router
	r := gin.Default()
	_ = r.SetTrustedProxies(nil)

	// CORS middleware
	r.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

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

	ga := handler.NewGoogleAuthHandler(userRepo, jwtSvc, cfg.FrontendURL)

	api := r.Group("/api")
	{
		authGroup := api.Group("/auth/google")
		authGroup.GET("/login", ga.Login)
		authGroup.GET("/callback", ga.Callback)

		// return user profile
		api.GET("/me", middleware.AuthRequired(jwtSvc), func(c *gin.Context) {
			ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
			defer cancel()

			uid := c.GetString(middleware.CtxUserID)

			u, err := userRepo.FindByID(ctx, uid)
			if err != nil {
				c.JSON(500, gin.H{"ok": false, "error": "db_failed"})
				return
			}

			c.JSON(200, gin.H{
				"ok":   true,
				"role": string(u.Role),
				"user": gin.H{
					"id":      u.ID.Hex(),
					"email":   u.Email,
					"name":    u.Name,
					"picture": u.Picture,
					"role":    string(u.Role),
				},
			})
		})

		api.GET("/admin/ping",
			middleware.AuthRequired(jwtSvc),
			middleware.RequireRole(model.RoleAdmin),
			func(c *gin.Context) {
				c.JSON(200, gin.H{
					"ok":    true,
					"admin": true,
				})
			},
		)

		// Seat lock routes
		st := api.Group("/showtimes/:showtimeId", middleware.AuthRequired(jwtSvc))
		st.POST("/seats/lock", seatLockHandler.Lock)
		st.DELETE("/seats/lock", seatLockHandler.Release)
		st.GET("/seats/locks", seatLockHandler.ListLocks)

		// Booking
		st.POST("/bookings/confirm", bookingHandler.Confirm)

	}

	// WebSocket
	r.GET("/ws/showtimes/:showtimeId/seats", seatWS.Seats)

	_ = r.Run(":" + cfg.Port)
}
