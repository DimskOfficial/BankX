// Path: internal/handlers/handlers.go
package handlers

import (
	"bank-api/internal/models"
	"bank-api/internal/services"
	"bank-api/pkg/utils"
	"errors"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"strconv"
)

// Handler holds dependencies for request handling.
type Handler struct {
	transactionService services.TransactionService
	authService        services.AuthService
	accountService     services.AccountService
}

// NewHandler creates a new Handler instance.
func NewHandler(ts services.TransactionService, as services.AuthService, acs services.AccountService) *Handler {
	return &Handler{
		transactionService: ts,
		authService:        as,
		accountService:     acs,
	}
}

// Custom error type to include context
type AppError struct {
	Code    int    `json:"-"`
	Message string `json:"message"`
	Details string `json:"details"`
	Err     error  `json:"-"`
}

func (e *AppError) Error() string {
	return fmt.Sprintf("AppError: %s (Code: %d, Details: %s, OriginalError: %v)", e.Message, e.Code, e.Details, e.Err)
}

// ErrorHandler handles errors from Fiber
func (h *Handler) ErrorHandler(c *fiber.Ctx, err error) error {
	fmt.Printf("Error: %v\n", err)

	code := fiber.StatusInternalServerError
	message := "Internal Server Error"
	details := ""

	var appErr *AppError
	if errors.As(err, &appErr) {
		code = appErr.Code
		message = appErr.Message
		details = appErr.Details
	} else if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	} else {
		details = err.Error()
	}

	return c.Status(code).JSON(fiber.Map{
		"error":   message,
		"details": details,
	})
}

// Register handler
func (h *Handler) Register(c *fiber.Ctx) error {
	var req models.AuthRequest
	if err := c.BodyParser(&req); err != nil {
		return &AppError{Code: fiber.StatusBadRequest, Message: "Invalid request format", Details: err.Error(), Err: err}
	}

	if err := h.authService.Register(req.Username, req.Password); err != nil {
		var appErr *services.AppError
		if errors.As(err, &appErr) {
			return appErr
		}
		return &AppError{Code: fiber.StatusInternalServerError, Message: "Registration failed", Details: err.Error(), Err: err}
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Registration successful"})
}

// Login handler
func (h *Handler) Login(c *fiber.Ctx) error {
	var req models.AuthRequest
	if err := c.BodyParser(&req); err != nil {
		return &AppError{Code: fiber.StatusBadRequest, Message: "Invalid request format", Details: err.Error(), Err: err}
	}

	token, err := h.authService.Login(req.Username, req.Password)
	if err != nil {
		var appErr *services.AppError
		if errors.As(err, &appErr) {
			return appErr
		}
		return &AppError{Code: fiber.StatusUnauthorized, Message: "Login failed", Details: err.Error(), Err: err}
	}

	return c.JSON(fiber.Map{"token": token})
}

// AuthMiddleware handles JWT authorization
func (h *Handler) AuthMiddleware(c *fiber.Ctx) error {
	if c.Method() == "OPTIONS" {
		return c.Next()
	}

	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return &AppError{Code: fiber.StatusUnauthorized, Message: "Missing token", Details: "Authorization header is empty"}
	}

	var token string
	if _, err := fmt.Sscanf(authHeader, "Bearer %s", &token); err != nil {
		return &AppError{Code: fiber.StatusUnauthorized, Message: "Invalid token format", Details: err.Error()}
	}

	claims, err := h.authService.ValidateToken(token)
	if err != nil {
		return &AppError{Code: fiber.StatusUnauthorized, Message: "Invalid token", Details: err.Error()}
	}

	c.Locals("user", claims)
	return c.Next()
}

// GetAccounts retrieves user accounts...
func (h *Handler) GetAccounts(c *fiber.Ctx) error {
	claims, ok := c.Locals("user").(*models.Claims)
	if !ok {
		return &AppError{Code: fiber.StatusInternalServerError, Message: "Failed to retrieve user claims", Details: "User claims were not of the expected type"}
	}

	accounts, err := h.accountService.GetAccounts(claims.UserID)
	if err != nil {
		var appErr *services.AppError

		if errors.As(err, &appErr) {
			return appErr
		}
		return &AppError{Code: fiber.StatusInternalServerError, Message: "Failed to retrieve accounts", Details: err.Error(), Err: err}
	}

	return c.JSON(accounts)
}

// Transfer handler for fund transfers between accounts.
func (h *Handler) Transfer(c *fiber.Ctx) error {
	claims, ok := c.Locals("user").(*models.Claims)
	if !ok {
		return &AppError{Code: fiber.StatusInternalServerError, Message: "Failed to retrieve user claims", Details: "User claims were not of the expected type"}
	}

	var req models.TransferRequest

	if err := c.BodyParser(&req); err != nil {
		return &AppError{Code: fiber.StatusBadRequest, Message: "Invalid request format", Details: err.Error(), Err: err}
	}

	transactionID, err := h.transactionService.ProcessTransfer(&req, claims)
	if err != nil {
		var appErr *services.AppError
		if errors.As(err, &appErr) {
			return appErr
		}
		return &AppError{Code: fiber.StatusBadRequest, Message: "Transfer failed", Details: err.Error(), Err: err}
	}

	return c.JSON(fiber.Map{"message": "Transfer successful", "transactionID": transactionID})
}

// Deposit handler for account deposits.
func (h *Handler) Deposit(c *fiber.Ctx) error {
	claims, ok := c.Locals("user").(*models.Claims)
	if !ok {
		return &AppError{Code: fiber.StatusInternalServerError, Message: "Failed to retrieve user claims", Details: "User claims were not of the expected type"}
	}

	accountID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return &AppError{Code: fiber.StatusBadRequest, Message: "Invalid account ID", Details: err.Error(), Err: err}
	}

	var req models.TransactionRequest
	if err := c.BodyParser(&req); err != nil {
		return &AppError{Code: fiber.StatusBadRequest, Message: "Invalid request format", Details: err.Error(), Err: err}
	}

	req.AccountID = accountID
	transactionID, err := h.transactionService.ProcessDeposit(&req, claims)
	if err != nil {
		var appErr *services.AppError
		if errors.As(err, &appErr) {
			return appErr
		}
		return &AppError{Code: fiber.StatusBadRequest, Message: "Deposit failed", Details: err.Error(), Err: err}
	}

	return c.JSON(fiber.Map{"message": "Deposit successful", "transactionID": transactionID})
}

// Withdraw handler for account withdrawals.
func (h *Handler) Withdraw(c *fiber.Ctx) error {
	claims, ok := c.Locals("user").(*models.Claims)
	if !ok {
		return &AppError{Code: fiber.StatusInternalServerError, Message: "Failed to retrieve user claims", Details: "User claims were not of the expected type"}
	}

	accountID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return &AppError{Code: fiber.StatusBadRequest, Message: "Invalid account ID", Details: err.Error(), Err: err}
	}

	var req models.TransactionRequest
	if err := c.BodyParser(&req); err != nil {
		return &AppError{Code: fiber.StatusBadRequest, Message: "Invalid request format", Details: err.Error(), Err: err}
	}

	req.AccountID = accountID
	transactionID, err := h.transactionService.ProcessWithdraw(&req, claims)
	if err != nil {
		var appErr *services.AppError
		if errors.As(err, &appErr) {
			return appErr
		}
		return &AppError{Code: fiber.StatusBadRequest, Message: "Withdrawal failed", Details: err.Error(), Err: err}
	}

	return c.JSON(fiber.Map{"message": "Withdrawal successful", "transactionID": transactionID})
}
