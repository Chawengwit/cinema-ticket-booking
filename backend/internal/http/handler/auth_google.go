package handler

import (
	"cinema/internal/auth"
	"cinema/internal/model"
	"cinema/internal/repo"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type GoogleAuthHandler struct {
	oauth       *oauth2.Config
	userRepo    *repo.UserRepo
	jwt         *auth.JWTService
	frontendURL string

	// NEW: admin email allowlist (lowercase)
	adminEmails map[string]struct{}
}

func NewGoogleAuthHandler(
	userRepo *repo.UserRepo,
	jwtSvc *auth.JWTService,
	frontendURL string,
	adminEmails []string, // from cfg.AdminEmails
) *GoogleAuthHandler {
	cfg := &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}

	m := make(map[string]struct{}, len(adminEmails))
	for _, e := range adminEmails {
		e = strings.ToLower(strings.TrimSpace(e))
		if e != "" {
			m[e] = struct{}{}
		}
	}

	return &GoogleAuthHandler{
		oauth:       cfg,
		userRepo:    userRepo,
		jwt:         jwtSvc,
		frontendURL: frontendURL,
		adminEmails: m,
	}
}

func (h *GoogleAuthHandler) Login(c *gin.Context) {
	state := "dev-state"
	url := h.oauth.AuthCodeURL(state, oauth2.AccessTypeOnline)
	c.Redirect(http.StatusFound, url)
}

func (h *GoogleAuthHandler) Callback(c *gin.Context) {
	code := c.Query("code")

	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"ok":    false,
			"error": "missing_code",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	tok, err := h.oauth.Exchange(ctx, code)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "exchange_failed"})
		return
	}

	// get userinfo
	client := h.oauth.Client(ctx, tok)
	res, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil || res.StatusCode != 200 {
		c.JSON(http.StatusBadRequest, gin.H{
			"ok":    false,
			"error": "userinfo_failed",
		})
		return
	}
	defer res.Body.Close()

	var u struct {
		ID      string `json:"id"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}

	if err := json.NewDecoder(res.Body).Decode(&u); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "decode_failed"})
		return
	}

	// upsert user
	user, err := h.userRepo.UpsertGoogleUser(ctx, u.ID, u.Email, u.Name, u.Picture)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"ok":    false,
			"error": "db_failed",
		})
		return
	}

	// NEW: decide role by ADMIN_EMAILS
	emailKey := strings.ToLower(strings.TrimSpace(u.Email))
	desiredRole := model.RoleUser
	if _, ok := h.adminEmails[emailKey]; ok {
		desiredRole = model.RoleAdmin
	}

	// if role changed, persist it and use updated user for JWT
	if user.Role != desiredRole {
		updated, err := h.userRepo.SetRoleByID(ctx, user.ID, desiredRole)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"ok":    false,
				"error": "set_role_failed",
			})
			return
		}
		user = updated
	}

	// sign jwt with latest role
	jwtToken, err := h.jwt.Sign(user.ID.Hex(), user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"ok":     false,
			"error":  "jwt_failed",
			"detail": err.Error(),
		})
		return
	}

	// redirect to FE with token
	c.Redirect(http.StatusFound, h.frontendURL+"/auth/callback?token="+jwtToken)
}
