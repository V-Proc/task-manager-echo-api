package controllers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/v-proc/task-manager-echo-api/database"
	"github.com/v-proc/task-manager-echo-api/middleware"
	"github.com/v-proc/task-manager-echo-api/models"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// TaskInput is the payload for create and update endpoints. Description is a
// pointer so update callers can distinguish "field absent" (leave as-is) from
// "field present and empty" (clear the value). Title and Status use zero-value
// guards instead.
type TaskInput struct {
	Title       string            `json:"title"`
	Description *string           `json:"description"`
	Status      models.TaskStatus `json:"status"`
}

// Create makes a new task owned by the authenticated user.
func Create(c echo.Context) error {
	uid := middleware.CurrentUserID(c)

	var in TaskInput
	if err := c.Bind(&in); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if in.Title == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "title is required")
	}
	if in.Status == "" {
		in.Status = models.StatusPending
	}
	if !isValidStatus(in.Status) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid status value")
	}

	task := models.Task{
		Title:       in.Title,
		Description: derefStr(in.Description),
		Status:      in.Status,
		UserID:      uid,
	}
	if err := database.DB.Create(&task).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, task)
}

// List returns all tasks belonging to the authenticated user. Optional
// `status` query param filters by status.
func List(c echo.Context) error {
	uid := middleware.CurrentUserID(c)

	q := database.DB.Model(&models.Task{}).Where("user_id = ?", uid)
	if status := c.QueryParam("status"); status != "" {
		if !isValidStatus(models.TaskStatus(status)) {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid status filter")
		}
		q = q.Where("status = ?", status)
	}

	var tasks []models.Task
	if err := q.Order("created_at DESC").Find(&tasks).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, tasks)
}

// Get returns a single task if it belongs to the authenticated user.
func Get(c echo.Context) error {
	uid := middleware.CurrentUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid task id")
	}

	var task models.Task
	err = database.DB.Where("id = ? AND user_id = ?", id, uid).First(&task).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return echo.NewHTTPError(http.StatusNotFound, "task not found")
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, task)
}

// Update partially updates a task owned by the authenticated user.
func Update(c echo.Context) error {
	uid := middleware.CurrentUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid task id")
	}

	var task models.Task
	err = database.DB.Where("id = ? AND user_id = ?", id, uid).First(&task).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return echo.NewHTTPError(http.StatusNotFound, "task not found")
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var in TaskInput
	if err := c.Bind(&in); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if in.Status != "" && !isValidStatus(in.Status) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid status value")
	}

	updates := map[string]interface{}{}
	if in.Title != "" {
		updates["title"] = in.Title
	}
	// Description is a pointer: nil means "leave unchanged", a non-nil value
	// (including the empty string) explicitly sets it — allowing callers to
	// clear the field.
	if in.Description != nil {
		updates["description"] = *in.Description
	}
	if in.Status != "" {
		updates["status"] = in.Status
	}

	if len(updates) == 0 {
		return c.JSON(http.StatusOK, task)
	}

	if err := database.DB.Model(&task).Updates(updates).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Reload to reflect DB defaults / triggers.
	if err := database.DB.First(&task, task.ID).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, task)
}

// Delete removes a task owned by the authenticated user.
func Delete(c echo.Context) error {
	uid := middleware.CurrentUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid task id")
	}

	tx := database.DB.Where("id = ? AND user_id = ?", id, uid).Delete(&models.Task{})
	if tx.Error != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, tx.Error.Error())
	}
	if tx.RowsAffected == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "task not found")
	}
	return c.NoContent(http.StatusNoContent)
}

func isValidStatus(s models.TaskStatus) bool {
	switch s {
	case models.StatusPending, models.StatusInProgress, models.StatusDone:
		return true
	}
	return false
}

// derefStr returns the string pointed to by p, or "" when p is nil.
func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
