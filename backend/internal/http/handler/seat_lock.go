package handler

import (
	"cinema/internal/http/middleware"
	"cinema/internal/seatlock"
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type SeatLockHandler struct {
	svc        *seatlock.Service
	ttlSeconds int
}

func NewSeatLockHandler(svc *seatlock.Service, ttlSeconds int) *SeatLockHandler {
	return &SeatLockHandler{svc: svc, ttlSeconds: ttlSeconds}
}

type lockReq struct {
	SeatIDs []string `json:"seat_ids"`
}

func (h *SeatLockHandler) Lock(c *gin.Context) {
	showtimeID := c.Param("showtimeId")
	owner := c.GetString(middleware.CtxUserID)

	var req lockReq
	if err := c.ShouldBindJSON(&req); err != nil || len(req.SeatIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "invalid_body"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	ok, conflicted, err := h.svc.LockSeats(ctx, showtimeID, req.SeatIDs, owner)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "lock_failed"})
		return
	}
	if !ok {
		c.JSON(http.StatusConflict, gin.H{
			"ok":         false,
			"error":      "seats_unavailable",
			"conflicted": []string{conflicted},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":          true,
		"locked":      req.SeatIDs,
		"ttl_seconds": h.ttlSeconds,
	})
}

func (h *SeatLockHandler) Release(c *gin.Context) {
	showtimeID := c.Param("showtimeId")
	owner := c.GetString(middleware.CtxUserID)

	var req lockReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "invalid_body"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	if err := h.svc.ReleaseSeats(ctx, showtimeID, req.SeatIDs, owner); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "release_failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "released": req.SeatIDs})
}
