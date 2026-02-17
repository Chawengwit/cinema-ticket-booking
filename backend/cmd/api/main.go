package main

import (
	"cinema/internal/audit"
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

	rootCtx := context.Background()

	// connect mongo
	mongoConn, err := db.ConnectMongo(rootCtx, cfg.MongoURI)
	if err != nil {
		panic(err)
	}
	defer func() { _ = mongoConn.Client.Disconnect(context.Background()) }()

	// connect redis
	redisClient, err := cache.ConnectRedis(rootCtx, cfg.RedisAddr)
	if err != nil {
		panic(err)
	}
	defer func() { _ = redisClient.Close() }()

	// services
	jwtSvc := auth.NewJWTService(cfg.JWTSecret)

	// repos
	userRepo := repo.NewUserRepo(mongoConn.DB)
	auditRepo := repo.NewAuditRepo(mongoConn.DB)
	bookingRepo := repo.NewBookingRepo(mongoConn.DB)

	// background workers
	go audit.Run(rootCtx, redisClient, auditRepo)
	go seatlock.StartTimeoutSweeper(rootCtx, redisClient)

	// WebSocket handler
	seatWS := handler.NewSeatWSHandler(redisClient, jwtSvc)

	// SeatLock service + handler
	seatTTL := time.Duration(cfg.SeatLockTTLSeconds) * time.Second
	seatLockSvc := seatlock.New(redisClient, seatTTL)
	seatLockHandler := handler.NewSeatLockHandler(seatLockSvc, cfg.SeatLockTTLSeconds)

	// Booking handler
	bookingHandler := handler.NewBookingHandler(seatLockSvc, bookingRepo, redisClient)

	// Admin handlers
	adminBookingHandler := handler.NewAdminBookingHandler(bookingRepo)
	adminAuditHandler := handler.NewAdminAuditHandler(auditRepo)

	// Google OAuth handler (ADMIN_EMAILS integrated via cfg.AdminEmails)
	ga := handler.NewGoogleAuthHandler(userRepo, jwtSvc, cfg.FrontendURL, cfg.AdminEmails)

	// router
	r := gin.Default()
	_ = r.SetTrustedProxies(nil)

	// CORS
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

	// API
	api := r.Group("/api")
	{
		// Auth
		authGroup := api.Group("/auth/google")
		authGroup.GET("/login", ga.Login)
		authGroup.GET("/callback", ga.Callback)

		// Me
		api.GET("/me", middleware.AuthRequired(jwtSvc), func(c *gin.Context) {
			ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
			defer cancel()

			uid := c.GetString(middleware.CtxUserID)

			u, err := userRepo.FindByID(ctx, uid)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "db_failed"})
				return
			}

			c.JSON(http.StatusOK, gin.H{
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

		// Admin (guard ด้วย role=ADMIN)
		admin := api.Group("/admin",
			middleware.AuthRequired(jwtSvc),
			middleware.RequireRole(model.RoleAdmin),
		)
		{
			admin.GET("/bookings", adminBookingHandler.List)
			admin.GET("/audit", adminAuditHandler.List)
			admin.GET("/ping", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"ok": true, "admin": true})
			})
		}

		// Showtime scoped routes
		st := api.Group("/showtimes/:showtimeId", middleware.AuthRequired(jwtSvc))
		{
			// Seat lock
			st.POST("/seats/lock", seatLockHandler.Lock)
			st.DELETE("/seats/lock", seatLockHandler.Release)
			st.GET("/seats/locks", seatLockHandler.ListLocks)

			// Booking confirm
			st.POST("/bookings/confirm", bookingHandler.Confirm)
		}
	}

	// WebSocket
	r.GET("/ws/showtimes/:showtimeId/seats", seatWS.Seats)

	_ = r.Run(":" + cfg.Port)
}
