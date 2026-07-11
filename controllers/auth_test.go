//go:build !(js && wasm)

package controllers_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/v-proc/task-manager-echo-api/controllers"
	"github.com/v-proc/task-manager-echo-api/database"
	"github.com/v-proc/task-manager-echo-api/internal/echotest"
	"github.com/v-proc/task-manager-echo-api/internal/testutil"
	"github.com/v-proc/task-manager-echo-api/models"
)

func TestRegister_CreatesUserAndReturnsToken(t *testing.T) {
	testutil.SetupDB(t)
	cfg := testutil.Config()

	rec := echotest.CallJSON(t, http.MethodPost, "/api/auth/register",
		`{"name":"Alice","email":"alice@example.com","password":"secret123"}`,
		controllers.Register(cfg))

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		User  models.User `json:"user"`
		Token string      `json:"token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.User.Email != "alice@example.com" {
		t.Errorf("email = %q", resp.User.Email)
	}
	if resp.User.Password != "" {
		t.Errorf("password hash leaked: %q", resp.User.Password)
	}
	if resp.Token == "" {
		t.Error("missing token")
	}
}

func TestRegister_DuplicateEmail_Conflict(t *testing.T) {
	testutil.SetupDB(t)
	cfg := testutil.Config()

	body := `{"name":"Bob","email":"bob@example.com","password":"secret123"}`
	if echotest.CallJSON(t, http.MethodPost, "/api/auth/register", body, controllers.Register(cfg)).Code != http.StatusCreated {
		t.Fatal("first registration failed")
	}

	rec := echotest.CallJSON(t, http.MethodPost, "/api/auth/register", body, controllers.Register(cfg))
	if rec.Code != http.StatusConflict {
		t.Errorf("duplicate status = %d, want 409", rec.Code)
	}
}

func TestRegister_ShortPassword_BadRequest(t *testing.T) {
	testutil.SetupDB(t)
	cfg := testutil.Config()

	rec := echotest.CallJSON(t, http.MethodPost, "/api/auth/register",
		`{"name":"C","email":"c@example.com","password":"abc"}`,
		controllers.Register(cfg))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestRegister_BadBody_BadRequest(t *testing.T) {
	testutil.SetupDB(t)
	cfg := testutil.Config()
	rec := echotest.CallJSON(t, http.MethodPost, "/api/auth/register", `{not-json`, controllers.Register(cfg))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestLogin_ValidCredentials_ReturnsToken(t *testing.T) {
	testutil.SetupDB(t)
	cfg := testutil.Config()

	// Seed a user via the register handler so the password is hashed correctly.
	echotest.CallJSON(t, http.MethodPost, "/api/auth/register",
		`{"name":"D","email":"d@example.com","password":"secret123"}`, controllers.Register(cfg))

	rec := echotest.CallJSON(t, http.MethodPost, "/api/auth/login",
		`{"email":"d@example.com","password":"secret123"}`, controllers.Login(cfg))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Token == "" {
		t.Error("missing token on login")
	}
}

func TestLogin_WrongPassword_Unauthorized(t *testing.T) {
	testutil.SetupDB(t)
	cfg := testutil.Config()

	echotest.CallJSON(t, http.MethodPost, "/api/auth/register",
		`{"name":"E","email":"e@example.com","password":"secret123"}`, controllers.Register(cfg))

	rec := echotest.CallJSON(t, http.MethodPost, "/api/auth/login",
		`{"email":"e@example.com","password":"wrong"}`, controllers.Login(cfg))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestLogin_UnknownEmail_Unauthorized(t *testing.T) {
	testutil.SetupDB(t)
	cfg := testutil.Config()
	rec := echotest.CallJSON(t, http.MethodPost, "/api/auth/login",
		`{"email":"nope@example.com","password":"x"}`, controllers.Login(cfg))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestMe_ReturnsCurrentUser(t *testing.T) {
	testutil.SetupDB(t)

	// Seed a user directly so we know its ID.
	user := models.User{Name: "F", Email: "f@example.com", Password: "hashed"}
	if err := database.DB.Create(&user).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	rec := echotest.CallAuthed(t, http.MethodGet, "/api/me", "", user.ID, controllers.Me)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var got models.User
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.ID != user.ID || got.Email != user.Email {
		t.Errorf("got = %+v", got)
	}
}

func TestMe_MissingUser_NotFound(t *testing.T) {
	testutil.SetupDB(t)
	rec := echotest.CallAuthed(t, http.MethodGet, "/api/me", "", 9999, controllers.Me)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}
