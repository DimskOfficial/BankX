// Path: internal/services/auth_service.go
package services

import (
	"bank-api/internal/models"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

// AuthService handles user authentication and registration.
type AuthService interface {
	Register(username, password string) error
	Login(username, password string) (string, error)
	ValidateToken(tokenString string) (*models.Claims, error)
}

type authService struct {
	db     *sql.DB
	jwtKey string
}

// NewAuthService creates a new AuthService.
func NewAuthService(db *sql.DB, jwtSecret string) AuthService {
	return &authService{
		db:     db,
		jwtKey: jwtSecret,
	}
}

// Register registers a new user.
func (s *authService) Register(username, password string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return &AppError{Code: 500, Message: "Failed to start transaction", Details: err.Error(), Err: err}
	}
	defer tx.Rollback() // Ensure rollback on failure.

	// Check if user already exists.
	var exists bool
	err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username = ?)", username).Scan(&exists)
	if err != nil {
		return &AppError{Code: 500, Message: "Failed to check user existence", Details: err.Error(), Err: err}
	}
	if exists {
		return &AppError{Code: 400, Message: "User already exists", Details: fmt.Sprintf("username: %s", username)}
	}

	// Hash the password.
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return &AppError{Code: 500, Message: "Failed to hash password", Details: err.Error(), Err: err}
	}

	// Insert the new user.
	res, err := tx.Exec("INSERT INTO users (username, password) VALUES (?, ?)", username, hashedPassword)
	if err != nil {
		return &AppError{Code: 500, Message: "Failed to insert user", Details: err.Error(), Err: err}
	}

	userID, err := res.LastInsertId()
	if err != nil {
		return &AppError{Code: 500, Message: "Failed to get last inserted ID", Details: err.Error(), Err: err}
	}

	// Create a default account for the user.
	initialHash := CalculateBalanceHash(0, int(userID), s.jwtKey) // Use consistent hashing
	_, err = tx.Exec("INSERT INTO accounts (user_id, balance, balance_hash) VALUES (?, ?, ?)", userID, 0, initialHash)
	if err != nil {
		return &AppError{Code: 500, Message: "Failed to create initial account", Details: err.Error(), Err: err}
	}

	if err := tx.Commit(); err != nil {
		return &AppError{Code: 500, Message: "Failed to commit transaction", Details: err.Error(), Err: err}
	}

	return nil
}

// Login authenticates a user and returns a JWT.
func (s *authService) Login(username, password string) (string, error) {
	var user models.User
	err := s.db.QueryRow("SELECT id, password FROM users WHERE username = ?", username).Scan(&user.ID, &user.Password)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", &AppError{Code: 401, Message: "Invalid credentials", Details: "User not found"}
		}
		return "", &AppError{Code: 500, Message: "Failed to query user", Details: err.Error(), Err: err}
	}

	// Check password.
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", &AppError{Code: 401, Message: "Invalid credentials", Details: "Incorrect password"}
	}

	// Create JWT claims.
	claims := &models.Claims{
		UserID: user.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "bank-api",
		},
	}

	// Create and sign the token.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.jwtKey))
	if err != nil {
		return "", &AppError{Code: 500, Message: "Failed to sign token", Details: err.Error(), Err: err}
	}

	return tokenString, nil
}

// ValidateToken validates a JWT and returns the claims.
func (s *authService) ValidateToken(tokenString string) (*models.Claims, error) {
	claims := &models.Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.jwtKey), nil
	})

	if err != nil {
		// Distinguish between different parsing errors for better diagnostics
		if ve, ok := err.(*jwt.ValidationError); ok {
			if ve.Errors&jwt.ValidationErrorMalformed != 0 {
				return nil, &AppError{Code: 401, Message: "Invalid token", Details: "Malformed token"}
			} else if ve.Errors&(jwt.ValidationErrorExpired|jwt.ValidationErrorNotValidYet) != 0 {
				return nil, &AppError{Code: 401, Message: "Invalid token", Details: "Token expired or not yet valid"}
			}
		}
		return nil, &AppError{Code: 401, Message: "Invalid token", Details: err.Error(), Err: err}
	}

	if !token.Valid {
		return nil, &AppError{Code: 401, Message: "Invalid token", Details: "Token is not valid"}
	}

	return claims, nil
}
