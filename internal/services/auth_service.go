// Path: internal/services/auth_service.go
package services

import (
	"bank-api/internal/models"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AuthService handles user authentication and registration.
type AuthService interface {
	Register(username, password string) error
	Login(username, password string) (string, error)
	ValidateToken(token string) (*models.Claims, error)
}

type authService struct {
	db     *gorm.DB
	jwtKey string
}

// NewAuthService creates a new AuthService.
func NewAuthService(db *gorm.DB, jwtSecret string) AuthService {
	return &authService{
		db:     db,
		jwtKey: jwtSecret,
	}
}

// Register registers a new user.
func (s *authService) Register(username, password string) error {
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Check if user already exists.
		var count int64
		err := tx.Model(&models.User{}).Where("username = ?", username).Count(&count).Error
		if err != nil {
			return &AppError{Code: 500, Message: "Failed to check user existence", Details: err.Error(), Err: err}
		}
		if count > 0 {
			return &AppError{Code: 400, Message: "User already exists", Details: fmt.Sprintf("username: %s", username)}
		}

		// Hash the password.
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return &AppError{Code: 500, Message: "Failed to hash password", Details: err.Error(), Err: err}
		}

		// Insert the new user.
		user := models.User{
			Username: username,
			Password: string(hashedPassword),
		}
		user.CreatedAt = time.Now().Format(time.RFC3339) // Set the CreatedAt field to the current time as a string
		if err := tx.Create(&user).Error; err != nil {
			return &AppError{Code: 500, Message: "Failed to insert user", Details: err.Error(), Err: err}
		}

		// Create a default account for the user.
		initialHash := CalculateBalanceHash(0, uint(user.ID), s.jwtKey) // Use consistent hashing
		account := models.Account{
			UserID:      user.ID,
			Balance:     0,
			BalanceHash: initialHash,
		}
		if err := tx.Create(&account).Error; err != nil {
			return &AppError{Code: 500, Message: "Failed to create initial account", Details: err.Error(), Err: err}
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

// Login authenticates a user and returns a JWT.
func (s *authService) Login(username, password string) (string, error) {
	var user models.User
	err := s.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
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
		UserID: uint(user.ID),
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
