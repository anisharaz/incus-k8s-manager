package handlers

import (
	"encoding/json"
	"strings"

	"github.com/anisharaz/incus-k8s-manager/be/internal/models"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserHandlers handles user endpoints. There is no authentication yet —
// users only exist to own cluster networks, clusters, and nodes.
type UserHandlers struct {
	db *gorm.DB
}

// NewUserHandlers creates a new user handler.
func NewUserHandlers(db *gorm.DB) *UserHandlers {
	return &UserHandlers{db: db}
}

// CreateUser creates a new user.
func (h *UserHandlers) CreateUser(c fiber.Ctx) error {
	var req models.CreateUserRequest
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error:   "invalid request body",
			Message: err.Error(),
			Code:    fiber.StatusBadRequest,
		})
	}

	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || len(req.Username) > 63 {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error:   "validation error",
			Message: "username must be between 1 and 63 characters",
			Code:    fiber.StatusBadRequest,
		})
	}

	var count int64
	h.db.Model(&models.User{}).Where("username = ?", req.Username).Count(&count)
	if count > 0 {
		return c.Status(fiber.StatusConflict).JSON(models.ErrorResponse{
			Error:   "user already exists",
			Message: "a user with this username already exists",
			Code:    fiber.StatusConflict,
		})
	}

	user := models.User{
		ID:       uuid.New().String(),
		Username: req.Username,
	}

	if err := h.db.Create(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error:   "database error",
			Message: err.Error(),
			Code:    fiber.StatusInternalServerError,
		})
	}

	return c.Status(fiber.StatusCreated).JSON(models.UserResponse{User: user})
}

// ListUsers returns all users.
func (h *UserHandlers) ListUsers(c fiber.Ctx) error {
	var users []models.User
	if err := h.db.Order("created_at DESC").Find(&users).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error:   "database error",
			Message: err.Error(),
			Code:    fiber.StatusInternalServerError,
		})
	}

	return c.JSON(models.UserListResponse{Users: users})
}

// GetUser returns a single user by ID.
func (h *UserHandlers) GetUser(c fiber.Ctx) error {
	var user models.User
	if err := h.db.Where("id = ?", c.Params("id")).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(models.ErrorResponse{
				Error:   "not found",
				Message: "user not found",
				Code:    fiber.StatusNotFound,
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error:   "database error",
			Message: err.Error(),
			Code:    fiber.StatusInternalServerError,
		})
	}

	return c.JSON(models.UserResponse{User: user})
}
