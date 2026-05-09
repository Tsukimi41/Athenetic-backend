package middleware

import (
	"net/http"
	"strings"

	"github.com/Tsukimi41/Athenetic-backend/internal/security"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// JWTAuth validates access tokens and injects user_id into the context.
func JWTAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		authHeader := strings.TrimSpace(c.Request().Header.Get("Authorization"))
		if authHeader == "" {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing authorization header"})
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid authorization header"})
		}

		claims, err := security.ParseToken(parts[1], security.TokenTypeAccess)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid or expired token"})
		}

		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid user in token"})
		}

		c.Set("user_id", userID)
		return next(c)
	}
}
