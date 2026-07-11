package controllers

import (
	"errors"
	"net/http"

	"github.com/v-proc/task-manager-echo-api/config"
	"github.com/v-proc/task-manager-echo-api/database"
	"github.com/v-proc/task-manager-echo-api/middleware"
	"github.com/v-proc/task-manager-echo-api/models"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// RegisterInput is the payload accepted by the registration endpoint.
type RegisterInput struct {
	Name     string `json:"name" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

// LoginInput is the payload accepted by the login endpoint.
type LoginInput struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// Register creates a new user account and returns a JWT immediately so the
// client can start making authenticated requests without a second round-trip.
func Register(cfg *config.Config) echo.HandlerFunc {
	return func(c echo.Context) error {
		var in RegisterInput
		if err := c.Bind(&in); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
		}
		if in.Name == "" || in.Email == "" || in.Password == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "name, email and password are required")
		}
		if len(in.Password) < 6 {
			return echo.NewHTTPError(http.StatusBadRequest, "password must be at least 6 characters")
		}

		var existing models.User
		err := database.DB.Where("email = ?", in.Email).First(&existing).Error
		if err == nil {
			return echo.NewHTTPError(http.StatusConflict, "email already registered")
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to hash password")
		}

		user := models.User{
			Name:     in.Name,
			Email:    in.Email,
			Password: string(hash),
		}
		if err := database.DB.Create(&user).Error; err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		token, err := middleware.IssueToken(cfg, user.ID, user.Email)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to issue token")
		}

		return c.JSON(http.StatusCreated, echo.Map{
			"user":  user,
			"token": token,
		})
	}
}

// Login verifies credentials and returns a JWT for the authenticated user.
func Login(cfg *config.Config) echo.HandlerFunc {
	return func(c echo.Context) error {
		var in LoginInput
		if err := c.Bind(&in); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
		}
		if in.Email == "" || in.Password == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "email and password are required")
		}

		var user models.User
		err := database.DB.Where("email = ?", in.Email).First(&user).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
		}
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(in.Password)); err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
		}

		token, err := middleware.IssueToken(cfg, user.ID, user.Email)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to issue token")
		}

		return c.JSON(http.StatusOK, echo.Map{
			"user":  user,
			"token": token,
		})
	}
}

// Me returns the currently authenticated user's profile.
func Me(c echo.Context) error {
	uid := middleware.CurrentUserID(c)
	var user models.User
	if err := database.DB.First(&user, uid).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "user not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, user)
}
