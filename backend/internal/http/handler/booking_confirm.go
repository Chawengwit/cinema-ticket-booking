package handler

import (
	"cinema/internal/http/middleware"
	"cinema/internal/model"
	"cinema/internal/repo"
	"cinema/internal/seatlock"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BookingHandler struct {
	seatLock *seatlock.Service
	bookings *repo.BookingRepo
	rdb      *redis.Client
}

func NewBookingHandler(seatLock *seatlock.Service, bookings *repo.BookingRepo, rdb *redis.Client) *BookingHandler {
	return &BookingHandler{seatLock: seatLock, bookings: bookings, rdb: rdb}
}

type confirmBookingReq struct {
	SeatIDs   []string `json:"seat_ids"`
	RequestID string   `json:"request_id"`
}

func bookingEventsChannel() string { return "booking-events" }

type BookingEvent struct {
	Type      string   `json:"type"` // "booking.success"
	BookingID string   `json:"booking_id"`
	Showtime  string   `json:"showtime_id"`
	UserID    string   `json:"user_id"`
	SeatIDs   []string `json:"seat_ids"`
	Amount    int64    `json:"amount"`
	Currency  string   `json:"currency"`
	At        int64    `json:"at"`
}

func (h *BookingHandler) Confirm(c *gin.Context) {
	showtimeID := c.Param("showtimeId")
	owner := c.GetString(middleware.CtxUserID)

	var req confirmBookingReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "invalid_body"})
		return
	}

	seatIDs, ok := normalizeSeatIDs(req.SeatIDs)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "invalid_seat_ids"})
		return
	}
	if strings.TrimSpace(req.RequestID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "missing_request_id"})
		return
	}

	uid, err := primitive.ObjectIDFromHex(owner)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "invalid_user"})
		return
	}

	// mock pricing: 100 THB per seat
	amount := int64(len(seatIDs)) * 100
	currency := "THB"

	booking := &model.Booking{
		ID:         primitive.NewObjectID(),
		ShowtimeID: showtimeID,
		UserID:     uid,
		SeatIDs:    seatIDs,
		Amount:     amount,
		Currency:   currency,
		RequestID:  strings.TrimSpace(req.RequestID),
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// 1) create PENDING
	if err := h.bookings.CreatePending(ctx, booking); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "db_create_failed"})
		return
	}

	// 2) mock payment success
	paymentRef := "mock_" + uuid.NewString()

	// 3) finalize Redis: LOCKED -> BOOKED (atomic)
	okBooked, conflicted, reason, err := h.seatLock.ConfirmSeatsBooked(
		ctx,
		showtimeID,
		seatIDs,
		owner,
		booking.RequestID,
		booking.ID.Hex(),
	)
	if err != nil {
		_ = h.bookings.MarkFailed(ctx, booking.ID)
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "confirm_failed"})
		return
	}
	if !okBooked {
		_ = h.bookings.MarkFailed(ctx, booking.ID)
		c.JSON(http.StatusConflict, gin.H{
			"ok":         false,
			"error":      "seats_unavailable",
			"reason":     reason,
			"conflicted": []string{conflicted},
		})
		return
	}

	// 4) mark BOOKED
	if err := h.bookings.MarkBooked(ctx, booking.ID, paymentRef); err != nil {
		_ = h.bookings.MarkFailed(ctx, booking.ID)
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "db_update_failed"})
		return
	}

	// 5) publish booking success (best-effort)
	ev := BookingEvent{
		Type:      "booking.success",
		BookingID: booking.ID.Hex(),
		Showtime:  showtimeID,
		UserID:    owner,
		SeatIDs:   seatIDs,
		Amount:    amount,
		Currency:  currency,
		At:        time.Now().Unix(),
	}
	if b, e := json.Marshal(ev); e == nil {
		_ = h.rdb.Publish(ctx, bookingEventsChannel(), b).Err()
	}

	c.JSON(http.StatusOK, gin.H{
		"ok": true,
		"booking": gin.H{
			"id":          booking.ID.Hex(),
			"showtime_id": showtimeID,
			"seat_ids":    seatIDs,
			"amount":      amount,
			"currency":    currency,
			"status":      model.BookingBooked,
			"payment_ref": paymentRef,
		},
	})
}
