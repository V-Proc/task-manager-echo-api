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

func TestCreate_PersistsTaskForUser(t *testing.T) {
	testutil.SetupDB(t)
	uid := testutil.SeedUser(t, "owner@example.com")

	rec := echotest.CallAuthed(t, http.MethodPost, "/api/tasks",
		`{"title":"Buy milk","description":"2L","status":"pending"}`, uid, controllers.Create)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var task models.Task
	if err := json.Unmarshal(rec.Body.Bytes(), &task); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if task.Title != "Buy milk" || task.UserID != uid || task.Status != models.StatusPending {
		t.Errorf("task = %+v", task)
	}

	// Verify it actually landed in the DB scoped to the user.
	var count int64
	database.DB.Model(&models.Task{}).Where("user_id = ?", uid).Count(&count)
	if count != 1 {
		t.Errorf("db count = %d, want 1", count)
	}
}

func TestCreate_MissingTitle_BadRequest(t *testing.T) {
	testutil.SetupDB(t)
	uid := testutil.SeedUser(t, "owner@example.com")
	rec := echotest.CallAuthed(t, http.MethodPost, "/api/tasks",
		`{"description":"no title"}`, uid, controllers.Create)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestCreate_InvalidStatus_BadRequest(t *testing.T) {
	testutil.SetupDB(t)
	uid := testutil.SeedUser(t, "owner@example.com")
	rec := echotest.CallAuthed(t, http.MethodPost, "/api/tasks",
		`{"title":"x","status":"weird"}`, uid, controllers.Create)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestCreate_DefaultsPendingStatus(t *testing.T) {
	testutil.SetupDB(t)
	uid := testutil.SeedUser(t, "owner@example.com")
	rec := echotest.CallAuthed(t, http.MethodPost, "/api/tasks",
		`{"title":"x"}`, uid, controllers.Create)
	var task models.Task
	json.Unmarshal(rec.Body.Bytes(), &task)
	if task.Status != models.StatusPending {
		t.Errorf("default status = %q, want pending", task.Status)
	}
}

func TestList_ReturnsOnlyOwnTasks(t *testing.T) {
	testutil.SetupDB(t)
	alice := testutil.SeedUser(t, "alice@example.com")
	bob := testutil.SeedUser(t, "bob@example.com")

	database.DB.Create(&models.Task{Title: "A1", UserID: alice})
	database.DB.Create(&models.Task{Title: "A2", UserID: alice})
	database.DB.Create(&models.Task{Title: "B1", UserID: bob})

	rec := echotest.CallAuthed(t, http.MethodGet, "/api/tasks", "", alice, controllers.List)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var tasks []models.Task
	json.Unmarshal(rec.Body.Bytes(), &tasks)
	if len(tasks) != 2 {
		t.Errorf("got %d tasks, want 2 (alice only)", len(tasks))
	}
	for _, tk := range tasks {
		if tk.UserID != alice {
			t.Errorf("leaked task from other user: %+v", tk)
		}
	}
}

func TestList_StatusFilter(t *testing.T) {
	testutil.SetupDB(t)
	uid := testutil.SeedUser(t, "owner@example.com")
	database.DB.Create(&models.Task{Title: "p1", UserID: uid, Status: models.StatusPending})
	database.DB.Create(&models.Task{Title: "d1", UserID: uid, Status: models.StatusDone})

	rec := echotest.CallAuthed(t, http.MethodGet, "/api/tasks?status=done", "", uid, controllers.List)
	var tasks []models.Task
	json.Unmarshal(rec.Body.Bytes(), &tasks)
	if len(tasks) != 1 || tasks[0].Title != "d1" {
		t.Errorf("filter result = %+v", tasks)
	}
}

func TestList_InvalidStatusFilter_BadRequest(t *testing.T) {
	testutil.SetupDB(t)
	uid := testutil.SeedUser(t, "owner@example.com")
	rec := echotest.CallAuthed(t, http.MethodGet, "/api/tasks?status=bogus", "", uid, controllers.List)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestGet_OwnedTask_ReturnsIt(t *testing.T) {
	testutil.SetupDB(t)
	uid := testutil.SeedUser(t, "owner@example.com")
	task := models.Task{Title: "T", UserID: uid}
	database.DB.Create(&task)

	pr := echotest.NewParamRequest(http.MethodGet, "", uid, task.ID)
	if err := controllers.Get(pr.Context); err != nil {
		pr.Context.Error(err)
	}
	if pr.Rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", pr.Rec.Code)
	}
}

func TestGet_ForeignTask_NotFound(t *testing.T) {
	testutil.SetupDB(t)
	alice := testutil.SeedUser(t, "alice@example.com")
	bob := testutil.SeedUser(t, "bob@example.com")
	task := models.Task{Title: "B-secret", UserID: bob}
	database.DB.Create(&task)

	pr := echotest.NewParamRequest(http.MethodGet, "", alice, task.ID)
	if err := controllers.Get(pr.Context); err != nil {
		pr.Context.Error(err)
	}
	if pr.Rec.Code != http.StatusNotFound {
		t.Errorf("alice accessing bob's task: status = %d, want 404", pr.Rec.Code)
	}
}

func TestUpdate_PatchesProvidedFieldsOnly(t *testing.T) {
	testutil.SetupDB(t)
	uid := testutil.SeedUser(t, "owner@example.com")
	task := models.Task{Title: "Old", Description: "orig", UserID: uid, Status: models.StatusPending}
	database.DB.Create(&task)

	pr := echotest.NewParamRequest(http.MethodPut, `{"status":"done"}`, uid, task.ID)
	if err := controllers.Update(pr.Context); err != nil {
		pr.Context.Error(err)
	}
	if pr.Rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", pr.Rec.Code, pr.Rec.Body.String())
	}

	var got models.Task
	json.Unmarshal(pr.Rec.Body.Bytes(), &got)
	if got.Title != "Old" {
		t.Errorf("title changed unexpectedly: %q", got.Title)
	}
	if got.Description != "orig" {
		t.Errorf("description changed: %q", got.Description)
	}
	if got.Status != models.StatusDone {
		t.Errorf("status = %q, want done", got.Status)
	}
}

func TestUpdate_InvalidStatus_BadRequest(t *testing.T) {
	testutil.SetupDB(t)
	uid := testutil.SeedUser(t, "owner@example.com")
	task := models.Task{Title: "T", UserID: uid}
	database.DB.Create(&task)

	pr := echotest.NewParamRequest(http.MethodPut, `{"status":"bogus"}`, uid, task.ID)
	if err := controllers.Update(pr.Context); err != nil {
		pr.Context.Error(err)
	}
	if pr.Rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", pr.Rec.Code)
	}
}

func TestDelete_RemovesOwnedTask(t *testing.T) {
	testutil.SetupDB(t)
	uid := testutil.SeedUser(t, "owner@example.com")
	task := models.Task{Title: "Bye", UserID: uid}
	database.DB.Create(&task)

	pr := echotest.NewParamRequest(http.MethodDelete, "", uid, task.ID)
	if err := controllers.Delete(pr.Context); err != nil {
		pr.Context.Error(err)
	}
	if pr.Rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", pr.Rec.Code)
	}

	// Soft-deleted, so a plain First() should return ErrRecordNotFound.
	var found models.Task
	err := database.DB.First(&found, task.ID).Error
	if err == nil {
		t.Error("task still present after delete")
	}
}

func TestDelete_ForeignTask_NotFound(t *testing.T) {
	testutil.SetupDB(t)
	alice := testutil.SeedUser(t, "alice@example.com")
	bob := testutil.SeedUser(t, "bob@example.com")
	task := models.Task{Title: "B", UserID: bob}
	database.DB.Create(&task)

	pr := echotest.NewParamRequest(http.MethodDelete, "", alice, task.ID)
	if err := controllers.Delete(pr.Context); err != nil {
		pr.Context.Error(err)
	}
	if pr.Rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", pr.Rec.Code)
	}
}

func TestGet_BadID_BadRequest(t *testing.T) {
	testutil.SetupDB(t)
	uid := testutil.SeedUser(t, "owner@example.com")

	// NewParamRequest takes a uint task ID, so use NewContext to inject a
	// non-numeric :id and exercise the bad-request path.
	c, rec := echotest.NewContext(http.MethodGet)
	c.SetParamNames("id")
	c.SetParamValues("not-a-number")
	c.Set("user_id", uid)
	if err := controllers.Get(c); err != nil {
		c.Error(err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}
