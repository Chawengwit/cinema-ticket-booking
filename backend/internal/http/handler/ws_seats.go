package handler

import (
	"cinema/internal/auth"
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

type SeatWSHandler struct {
	rdb    *redis.Client
	jwtSvc *auth.JWTService
}

func NewSeatWSHandler(rdb *redis.Client, jwtSvc *auth.JWTService) *SeatWSHandler {
	return &SeatWSHandler{rdb: rdb, jwtSvc: jwtSvc}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// ตอน dev ให้ผ่านก่อน (prod ควรเช็ค origin whitelist)
	CheckOrigin: func(r *http.Request) bool { return true },
}

func seatEventsChannel(showtimeID string) string {
	return "seat-events:" + showtimeID
}

// GET /ws/showtimes/:showtimeId/seats?token=JWT
func (h *SeatWSHandler) Seats(c *gin.Context) {
	showtimeID := c.Param("showtimeId")
	token := strings.TrimSpace(c.Query("token"))
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "missing_token"})
		return
	}

	// verify JWT
	_, err := h.jwtSvc.Verify(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "invalid_token"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// subscribe redis pubsub
	pubsub := h.rdb.Subscribe(ctx, seatEventsChannel(showtimeID))
	defer func() { _ = pubsub.Close() }()

	// ping/pong กันหลุดง่าย
	_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// read loop (แค่ไว้ detect disconnect)
	go func() {
		for {
			if _, _, e := conn.ReadMessage(); e != nil {
				cancel()
				return
			}
		}
	}()

	// optional: ส่ง hello
	_ = conn.WriteJSON(gin.H{"type": "hello", "showtime_id": showtimeID})

	ch := pubsub.Channel()

	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// ws ping
			_ = conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(2*time.Second))
		case msg := <-ch:
			// msg.Payload เป็น JSON string จาก seatlock.publish()
			_ = conn.WriteMessage(websocket.TextMessage, []byte(msg.Payload))
		}
	}
}
