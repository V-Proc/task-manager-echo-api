package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/v-proc/task-manager-echo-api/config"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

// Claims is the JWT payload we issue at login and verify on protected routes.
type Claims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// RequireAuth is an Echo middleware that validates a Bearer JWT and stores the
// parsed user ID on the echo.Context for downstream handlers.
func RequireAuth(cfg *config.Config) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			header := c.Request().Header.Get("Authorization")
			if header == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing authorization header")
			}

			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid authorization header")
			}

			claims := &Claims{}
			token, err := jwt.ParseWithClaims(parts[1], claims, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, echo.NewHTTPError(http.StatusUnauthorized, "unexpected signing method")
				}
				return []byte(cfg.JWTSecret), nil
			})
			if err != nil || !token.Valid {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired token")
			}

			c.Set("user_id", claims.UserID)
			c.Set("user_email", claims.Email)
			return next(c)
		}
	}
}

// CurrentUserID returns the authenticated user's ID stored by RequireAuth, or 0
// if the request was not authenticated.
func CurrentUserID(c echo.Context) uint {
	if v, ok := c.Get("user_id").(uint); ok {
		return v
	}
	return 0
}

// IssueToken builds and signs a new JWT for the given user. The expiry is taken
// from the application config.
func IssueToken(cfg *config.Config, userID uint, email string) (string, error) {
	claims := &Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(cfg.JWTExpireHrs) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWTSecret))
}
