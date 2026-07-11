// Package main holds the go-app component implementing the task manager UI.
//
// The component manages two views gated on authentication state:
//   - The auth view (login + register forms) shown when no token is stored.
//   - The tasks view (list + create + inline edit/delete) shown once logged in.
//
// The JWT token is persisted in browser localStorage so a reload keeps the
// session. All API calls use relative "/api/*" URLs; the dev server
// reverse-proxies those to the Echo backend, so no CORS configuration is
// needed.
package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/maxence-charriere/go-app/v9/pkg/app"
)

// Task mirrors the subset of the server-side models.Task payload the client uses.
type Task struct {
	ID          uint   `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

// authResponse is the JSON returned by /api/auth/register and /api/auth/login.
type authResponse struct {
	Token string `json:"token"`
}

const tokenKey = "tm.token"

// TaskApp is the root component. Public fields are bound to inputs via
// Compo.ValueTo and read on submit, so they intentionally live on the struct.
type TaskApp struct {
	app.Compo

	// Auth view state.
	loginEmail    string
	loginPassword string
	regName       string
	regEmail      string
	regPassword   string

	// Tasks view state.
	tasks         []Task
	newTitle      string
	newDesc       string
	newStatus     string
	editingID     uint
	editingTitle  string
	editingStatus string

	// Cross-cutting.
	token   string
	loading bool
	err     string
}

// OnMount restores any saved token and, if present, loads the user's tasks.
func (t *TaskApp) OnMount(ctx app.Context) {
	var stored string
	if err := ctx.LocalStorage().Get(tokenKey, &stored); err == nil {
		t.token = stored
	}
	if t.token != "" {
		t.loadTasks(ctx)
	}
}

// OnNav is required by the Composer interface but we don't branch per route —
// both "/" and "/login" render the same root component.
func (t *TaskApp) OnNav(ctx app.Context) {}

// ---- Auth handlers ---------------------------------------------------------

func (t *TaskApp) login(ctx app.Context, _ app.Event) {
	body := map[string]string{"email": t.loginEmail, "password": t.loginPassword}
	t.startLoading(ctx)
	go t.doAuth(ctx, http.MethodPost, "/api/auth/login", body, http.StatusOK)
}

func (t *TaskApp) register(ctx app.Context, _ app.Event) {
	body := map[string]string{
		"name":     t.regName,
		"email":    t.regEmail,
		"password": t.regPassword,
	}
	t.startLoading(ctx)
	go t.doAuth(ctx, http.MethodPost, "/api/auth/register", body, http.StatusCreated)
}

func (t *TaskApp) doAuth(ctx app.Context, method, path string, body map[string]string, wantStatus int) {
	resp, err := t.api(method, path, body)
	if err != nil {
		t.failOnUI(ctx, err.Error())
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != wantStatus {
		t.failOnUI(ctx, httpError(resp))
		return
	}
	var auth authResponse
	if err := json.NewDecoder(resp.Body).Decode(&auth); err != nil {
		t.failOnUI(ctx, err.Error())
		return
	}
	ctx.Dispatch(func(ctx app.Context) {
		t.token = auth.Token
		_ = ctx.LocalStorage().Set(tokenKey, t.token)
		t.loading = false
		t.loadTasks(ctx)
	})
}

func (t *TaskApp) logout(ctx app.Context, _ app.Event) {
	t.token = ""
	t.tasks = nil
	t.loginEmail = ""
	t.loginPassword = ""
	ctx.LocalStorage().Del(tokenKey)
	t.Update()
}

// ---- Task handlers ---------------------------------------------------------

func (t *TaskApp) loadTasks(ctx app.Context) {
	t.startLoading(ctx)
	go func() {
		resp, err := t.api(http.MethodGet, "/api/tasks", nil)
		if err != nil {
			t.failOnUI(ctx, err.Error())
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			if resp.StatusCode == http.StatusUnauthorized {
				// Token expired or invalid — drop the session and bounce to auth.
				ctx.Dispatch(func(ctx app.Context) {
					t.token = ""
					ctx.LocalStorage().Del(tokenKey)
					t.loading = false
					t.err = httpError(resp)
				})
				return
			}
			t.failOnUI(ctx, httpError(resp))
			return
		}
		var tasks []Task
		if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
			t.failOnUI(ctx, err.Error())
			return
		}
		ctx.Dispatch(func(_ app.Context) {
			t.tasks = tasks
			t.loading = false
			t.err = ""
		})
	}()
}

func (t *TaskApp) create(ctx app.Context, _ app.Event) {
	if strings.TrimSpace(t.newTitle) == "" {
		t.err = "title is required"
		t.Update()
		return
	}
	status := t.newStatus
	if status == "" {
		status = "pending"
	}
	body := map[string]string{
		"title":       t.newTitle,
		"description": t.newDesc,
		"status":      status,
	}
	go func() {
		resp, err := t.api(http.MethodPost, "/api/tasks", body)
		if err != nil {
			t.failOnUI(ctx, err.Error())
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.failOnUI(ctx, httpError(resp))
			return
		}
		ctx.Dispatch(func(ctx app.Context) {
			t.newTitle = ""
			t.newDesc = ""
			t.newStatus = ""
			t.loadTasks(ctx)
		})
	}()
}

func (t *TaskApp) saveEdit(ctx app.Context, _ app.Event) {
	if t.editingID == 0 {
		return
	}
	id := t.editingID
	body := map[string]string{"title": t.editingTitle, "status": t.editingStatus}
	go func() {
		resp, err := t.api(http.MethodPut, "/api/tasks/"+utoa(id), body)
		if err != nil {
			t.failOnUI(ctx, err.Error())
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.failOnUI(ctx, httpError(resp))
			return
		}
		ctx.Dispatch(func(ctx app.Context) {
			t.editingID = 0
			t.loadTasks(ctx)
		})
	}()
}

func (t *TaskApp) cancelEdit(_ app.Context, _ app.Event) {
	t.editingID = 0
	t.Update()
}

// remove is built per-row via a closure in render.go so the task ID is captured
// directly, avoiding fragile DOM traversal.
func (t *TaskApp) remove(ctx app.Context, id uint) {
	go func() {
		resp, err := t.api(http.MethodDelete, "/api/tasks/"+utoa(id), nil)
		if err != nil {
			t.failOnUI(ctx, err.Error())
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNoContent {
			t.failOnUI(ctx, httpError(resp))
			return
		}
		ctx.Dispatch(func(ctx app.Context) {
			t.loadTasks(ctx)
		})
	}()
}

// ---- Networking + state helpers --------------------------------------------

// api performs an authenticated JSON request. body may be nil for GET/DELETE.
func (t *TaskApp) api(method, path string, body interface{}) (*http.Response, error) {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		rdr = strings.NewReader(string(b))
	}
	req, err := http.NewRequest(method, path, rdr)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if t.token != "" {
		req.Header.Set("Authorization", "Bearer "+t.token)
	}
	return http.DefaultClient.Do(req)
}

// startLoading clears any error/info and shows the loading state on the UI
// goroutine. Event handlers already run on the UI goroutine, so this is safe
// to call directly.
func (t *TaskApp) startLoading(_ app.Context) {
	t.err = ""
	t.loading = true
	t.Update()
}

// failOnUI reports an error from a background goroutine, dispatching onto the
// UI goroutine so the re-render is safe.
func (t *TaskApp) failOnUI(ctx app.Context, msg string) {
	ctx.Dispatch(func(_ app.Context) {
		t.err = msg
		t.loading = false
	})
}

// httpError reads the response body and produces a readable message for a
// non-2xx HTTP response. Must be called from the background goroutine before
// dispatching state changes.
func httpError(resp *http.Response) string {
	b, _ := io.ReadAll(resp.Body)
	msg := strings.TrimSpace(string(b))
	if msg == "" {
		return "request failed: " + resp.Status
	}
	return "request failed (" + resp.Status + "): " + msg
}
