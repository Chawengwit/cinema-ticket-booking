package handler

import (
	"cinema/internal/repo"
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type AdminAuditHandler struct {
	auditRepo *repo.AuditRepo
}

func NewAdminAuditHandler(auditRepo *repo.AuditRepo) *AdminAuditHandler {
	return &AdminAuditHandler{auditRepo: auditRepo}
}

// GET /api/admin/audit?type=&showtime_id=&user_id=&booking_id=&from=&to=&limit=&skip=
func (h *AdminAuditHandler) List(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var f repo.AdminAuditFilter
	f.Type = c.Query("type")
	f.ShowtimeID = c.Query("showtime_id")
	f.UserID = c.Query("user_id")
	f.BookingID = c.Query("booking_id")

	if from := c.Query("from"); from != "" {
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

	items, total, err := h.auditRepo.FindAdmin(ctx, f)
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
