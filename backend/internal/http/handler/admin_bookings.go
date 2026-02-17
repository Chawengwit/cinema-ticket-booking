package handler

import (
	"cinema/internal/model"
	"cinema/internal/repo"
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AdminBookingHandler struct {
	bookingRepo *repo.BookingRepo
}

func NewAdminBookingHandler(bookingRepo *repo.BookingRepo) *AdminBookingHandler {
	return &AdminBookingHandler{bookingRepo: bookingRepo}
}

// GET /api/admin/bookings?showtime_id=&status=&user_id=&from=&to=&limit=&skip=
func (h *AdminBookingHandler) List(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var f repo.AdminBookingFilter
	f.ShowtimeID = c.Query("showtime_id")

	if st := c.Query("status"); st != "" {
		f.Status = model.BookingStatus(st)
	}

	if uid := c.Query("user_id"); uid != "" {
		oid, err := primitive.ObjectIDFromHex(uid)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "invalid_user_id"})
			return
		}
		f.UserID = oid
		f.HasUserID = true
	}

	if from := c.Query("from"); from != "" {
		// รองรับ RFC3339 เช่น 2026-02-18T00:00:00Z
		t, err := time.Parse(time.RFC3339, from)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "invalid_from"})
			return
		}
		f.From = &t
	}
	if to := c.Query("to"); to != "" {
		t, err := time.Parse(time.RFC3339, to)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "invalid_to"})
			return
		}
		f.To = &t
	}

	if v := c.Query("limit"); v != "" {
		n, _ := strconv.ParseInt(v, 10, 64)
		f.Limit = n
	}
	if v := c.Query("skip"); v != "" {
		n, _ := strconv.ParseInt(v, 10, 64)
		f.Skip = n
	}

	items, total, err := h.bookingRepo.FindAdmin(ctx, f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "db_failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":    true,
		"total": total,
		"items": items,
	})
}
