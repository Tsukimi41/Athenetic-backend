package handlers

import (
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/Tsukimi41/Athenetic-backend/internal/database"
	"github.com/Tsukimi41/Athenetic-backend/internal/models"
	"github.com/Tsukimi41/Athenetic-backend/internal/security"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type SignupRequest struct {
	Email      string  `json:"email"`
	Password   string  `json:"password"`
	Name       string  `json:"name"`
	BodyWeight float64 `json:"body_weight_kg"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int    `json:"expires_in"`
}

type MeResponse struct {
	ID         uuid.UUID `json:"id"`
	Email      string    `json:"email"`
	Name       string    `json:"name"`
	BodyWeight float64   `json:"body_weight_kg"`
	CreatedAt  time.Time `json:"created_at"`
}

type UpdateMeRequest struct {
	Name       *string  `json:"name"`
	BodyWeight *float64 `json:"body_weight_kg"`
}

func Signup(c echo.Context) error {
	var req SignupRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	email := normalizeEmail(req.Email)
	if email == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "email is required"})
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid email format"})
	}

	if len(req.Password) < 12 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "password must be at least 12 characters"})
	}

	if req.BodyWeight < 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "body_weight_kg must be positive"})
	}

	db := database.DB

	var existing models.User
	result := db.Where("LOWER(email) = LOWER(?)", email).First(&existing)
	if result.Error == nil {
		return c.JSON(http.StatusConflict, map[string]string{"error": "email already registered"})
	}
	if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "database error"})
	}

	passwordHash, err := security.HashPassword(req.Password)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to hash password"})
	}

	user := models.User{
		Email:        email,
		PasswordHash: passwordHash,
		Name:         strings.TrimSpace(req.Name),
		BodyWeight:   req.BodyWeight,
	}

	if err := db.Create(&user).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create user"})
	}

	accessToken, refreshToken, expiresIn, err := issueTokens(db, user.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to issue tokens"})
	}

	setRefreshCookie(c, refreshToken, security.RefreshTokenTTL())

	return c.JSON(http.StatusCreated, AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
	})
}

func Login(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	email := normalizeEmail(req.Email)
	if email == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "email and password are required"})
	}

	db := database.DB
	var user models.User
	result := db.Where("LOWER(email) = LOWER(?)", email).First(&user)
	if result.Error != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
	}

	if !security.CheckPassword(user.PasswordHash, req.Password) {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
	}

	accessToken, refreshToken, expiresIn, err := issueTokens(db, user.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to issue tokens"})
	}

	setRefreshCookie(c, refreshToken, security.RefreshTokenTTL())

	return c.JSON(http.StatusOK, AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
	})
}

func Refresh(c echo.Context) error {
	var req RefreshRequest
	_ = c.Bind(&req)

	refreshToken := strings.TrimSpace(req.RefreshToken)
	if refreshToken == "" {
		cookie, err := c.Cookie("refresh_token")
		if err == nil {
			refreshToken = cookie.Value
		}
	}
	if refreshToken == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "refresh token required"})
	}

	claims, err := security.ParseToken(refreshToken, security.TokenTypeRefresh)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid refresh token"})
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid refresh token"})
	}

	db := database.DB

	var stored models.RefreshToken
	result := db.Where("user_id = ? AND token_hash = ? AND revoked_at IS NULL AND expires_at > ?", userID, security.HashToken(refreshToken), time.Now().UTC()).First(&stored)
	if result.Error != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "refresh token revoked"})
	}

	// Rotate token
	now := time.Now().UTC()
	stored.RevokedAt = &now
	if err := db.Save(&stored).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to rotate token"})
	}

	accessToken, newRefreshToken, expiresIn, err := issueTokens(db, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to issue tokens"})
	}

	setRefreshCookie(c, newRefreshToken, security.RefreshTokenTTL())

	return c.JSON(http.StatusOK, AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    expiresIn,
	})
}

func Logout(c echo.Context) error {
	var req RefreshRequest
	_ = c.Bind(&req)

	refreshToken := strings.TrimSpace(req.RefreshToken)
	if refreshToken == "" {
		cookie, err := c.Cookie("refresh_token")
		if err == nil {
			refreshToken = cookie.Value
		}
	}
	if refreshToken == "" {
		clearRefreshCookie(c)
		return c.JSON(http.StatusOK, map[string]string{"message": "logged out"})
	}

	claims, err := security.ParseToken(refreshToken, security.TokenTypeRefresh)
	if err != nil {
		clearRefreshCookie(c)
		return c.JSON(http.StatusOK, map[string]string{"message": "logged out"})
	}

	userID, err := uuid.Parse(claims.UserID)
	if err == nil {
		db := database.DB
		now := time.Now().UTC()
		_ = db.Model(&models.RefreshToken{}).
			Where("user_id = ? AND token_hash = ?", userID, security.HashToken(refreshToken)).
			Update("revoked_at", &now).Error
	}

	clearRefreshCookie(c)
	return c.JSON(http.StatusOK, map[string]string{"message": "logged out"})
}

func Me(c echo.Context) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	db := database.DB
	var user models.User
	if err := db.First(&user, "id = ?", userID).Error; err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "user not found"})
	}

	return c.JSON(http.StatusOK, MeResponse{
		ID:         user.ID,
		Email:      user.Email,
		Name:       user.Name,
		BodyWeight: user.BodyWeight,
		CreatedAt:  user.CreatedAt,
	})
}

func UpdateMe(c echo.Context) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	var req UpdateMeRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["name"] = strings.TrimSpace(*req.Name)
	}
	if req.BodyWeight != nil {
		if *req.BodyWeight < 0 {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "body_weight_kg must be positive"})
		}
		updates["body_weight"] = *req.BodyWeight
	}
	if len(updates) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "no fields to update"})
	}

	db := database.DB
	if err := db.Model(&models.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update profile"})
	}

	var user models.User
	if err := db.First(&user, "id = ?", userID).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to load profile"})
	}

	return c.JSON(http.StatusOK, MeResponse{
		ID:         user.ID,
		Email:      user.Email,
		Name:       user.Name,
		BodyWeight: user.BodyWeight,
		CreatedAt:  user.CreatedAt,
	})
}

func issueTokens(db *gorm.DB, userID uuid.UUID) (string, string, int, error) {
	accessToken, accessExpiresAt, _, err := security.GenerateToken(userID, security.TokenTypeAccess, security.AccessTokenTTL())
	if err != nil {
		return "", "", 0, err
	}

	refreshToken, refreshExpiresAt, refreshTokenID, err := security.GenerateToken(userID, security.TokenTypeRefresh, security.RefreshTokenTTL())
	if err != nil {
		return "", "", 0, err
	}

	record := models.RefreshToken{
		UserID:    userID,
		TokenID:   refreshTokenID,
		TokenHash: security.HashToken(refreshToken),
		ExpiresAt: refreshExpiresAt,
	}

	if err := db.Create(&record).Error; err != nil {
		return "", "", 0, err
	}

	expiresIn := int(time.Until(accessExpiresAt).Seconds())
	return accessToken, refreshToken, expiresIn, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func setRefreshCookie(c echo.Context, refreshToken string, ttl time.Duration) {
	secure := strings.ToLower(strings.TrimSpace(c.Request().Header.Get("X-Forwarded-Proto"))) == "https"
	if strings.ToLower(strings.TrimSpace(c.Request().Header.Get("X-Forwarded-Ssl"))) == "on" {
		secure = true
	}
	if strings.ToLower(strings.TrimSpace(c.Request().Header.Get("X-Forwarded-Protocol"))) == "https" {
		secure = true
	}
	if strings.ToLower(strings.TrimSpace(c.Request().Header.Get("X-Forwarded-Scheme"))) == "https" {
		secure = true
	}
	if strings.ToLower(strings.TrimSpace(c.Request().Header.Get("Forwarded"))) != "" {
		secure = secure || strings.Contains(strings.ToLower(c.Request().Header.Get("Forwarded")), "proto=https")
	}

	cookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/api/v1/auth",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().UTC().Add(ttl),
	}
	c.SetCookie(cookie)
}

func clearRefreshCookie(c echo.Context) {
	cookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/api/v1/auth",
		HttpOnly: true,
		Secure:   strings.ToLower(strings.TrimSpace(c.Request().Header.Get("X-Forwarded-Proto"))) == "https",
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	}
	c.SetCookie(cookie)
}
