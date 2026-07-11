//go:build !(js && wasm)

package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/v-proc/task-manager-echo-api/internal/testutil"
	"github.com/v-proc/task-manager-echo-api/middleware"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

// parseToken decodes a token signed with the given secret into a Claims struct.
func parseToken(secret, token string) (*jwt.Token, *middleware.Claims, error) {
	claims := &middleware.Claims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	return parsed, claims, err
}

func TestIssueToken_ParsesBackWithSameClaims(t *testing.T) {
	cfg := testutil.Config()

	token, err := middleware.IssueToken(cfg, 42, "user@example.com")
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	if token == "" {
		t.Fatal("empty token returned")
	}

	parsed, claims, err := parseToken(cfg.JWTSecret, token)
	if err != nil || !parsed.Valid {
		t.Fatalf("parse token: %v", err)
	}
	if claims.UserID != 42 {
		t.Errorf("UserID = %d, want 42", claims.UserID)
	}
	if claims.Email != "user@example.com" {
		t.Errorf("Email = %q", claims.Email)
	}
}

func TestRequireAuth_MissingHeader_Rejects(t *testing.T) {
	cfg := testutil.Config()
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := middleware.RequireAuth(cfg)(echo.HandlerFunc(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}))
	err := h(c)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	he, ok := err.(*echo.HTTPError)
	if !ok || he.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %v", err)
	}
}

func TestRequireAuth_ValidBearer_SetsUserID(t *testing.T) {
	cfg := testutil.Config()
	e := echo.New()
	token, err := middleware.IssueToken(cfg, 7, "x@example.com")
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	var capturedID uint
	h := middleware.RequireAuth(cfg)(echo.HandlerFunc(func(c echo.Context) error {
		capturedID = middleware.CurrentUserID(c)
		return c.String(http.StatusOK, "ok")
	}))
	if err := h(c); err != nil {
		t.Fatalf("handler: %v", err)
	}
	if capturedID != 7 {
		t.Errorf("CurrentUserID = %d, want 7", capturedID)
	}
}

func TestRequireAuth_WrongSigningKey_Rejects(t *testing.T) {
	cfg := testutil.Config()
	other := *cfg
	other.JWTSecret = "different-secret"
	token, _ := middleware.IssueToken(&other, 1, "a@b.com")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := middleware.RequireAuth(cfg)(echo.HandlerFunc(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}))
	err := h(c)
	he, ok := err.(*echo.HTTPError)
	if !ok || he.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for tampered token, got %v", err)
	}
}

func TestRequireAuth_MalformedHeader_Rejects(t *testing.T) {
	cfg := testutil.Config()
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set("Authorization", "Token abc") // wrong scheme
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := middleware.RequireAuth(cfg)(echo.HandlerFunc(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}))
	if err := h(c); err == nil {
		t.Fatal("expected error for bad scheme")
	}
}

func TestCurrentUserID_ReturnsZeroWhenUnset(t *testing.T) {
	e := echo.New()
	c := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
	if id := middleware.CurrentUserID(c); id != 0 {
		t.Errorf("CurrentUserID = %d, want 0", id)
	}
}
