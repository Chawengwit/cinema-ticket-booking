package handler

import (
	"cinema/internal/http/middleware"
	"cinema/internal/seatlock"
	"context"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

var seatIDRe = regexp.MustCompile(`^[A-Z]{1,3}[0-9]{1,3}$`)

// normalize:
// - trim + uppercase
// - dedupe
// - validate format
// - sort for stable response
func normalizeSeatIDs(in []string) ([]string, bool) {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))

	for _, s := range in {
		s = strings.TrimSpace(strings.ToUpper(s))
		if s == "" {
			continue
		}
		if !seatIDRe.MatchString(s) {
			return nil, false
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}

	sort.Strings(out)
	return out, len(out) > 0
}

func (h *SeatLockHandler) Lock(c *gin.Context) {
	showtimeID := c.Param("showtimeId")
	owner := c.GetString(middleware.CtxUserID)

	var req lockReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "invalid_body"})
		return
	}

	seatIDs, ok := normalizeSeatIDs(req.SeatIDs)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "invalid_seat_ids"})
		return
	}

	// request-id (optional header, otherwise generate)
	rid := strings.TrimSpace(c.GetHeader("X-Request-Id"))
	if rid == "" {
		rid = uuid.NewString()
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	okLock, conflicted, err := h.svc.LockSeats(ctx, showtimeID, seatIDs, owner, rid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "lock_failed"})
		return
	}
	if !okLock {
		c.JSON(http.StatusConflict, gin.H{
			"ok":         false,
			"error":      "seats_unavailable",
			"conflicted": []string{conflicted},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":          true,
		"locked":      seatIDs,
		"ttl_seconds": h.ttlSeconds,
		"request_id":  rid,
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

	seatIDs, ok := normalizeSeatIDs(req.SeatIDs)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "invalid_seat_ids"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	if err := h.svc.ReleaseSeats(ctx, showtimeID, seatIDs, owner); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "release_failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "released": seatIDs})
}

// Debug/dev endpoint
func (h *SeatLockHandler) ListLocks(c *gin.Context) {
	showtimeID := c.Param("showtimeId")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	locks, err := h.svc.ListLocks(ctx, showtimeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "list_failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":          true,
		"showtime_id": showtimeID,
		"locks":       locks,
	})
}

func (h *SeatLockHandler) SeatState(c *gin.Context) {
	showtimeID := c.Param("showtimeId")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	locks, err := h.svc.ListLocks(ctx, showtimeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "list_failed"})
		return
	}

	booked, err := h.svc.ListBookedSeats(ctx, showtimeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "list_failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":          true,
		"showtime_id": showtimeID,
		"locks":       locks,
		"booked":      booked,
	})
}
