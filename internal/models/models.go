// Path: internal/models/models.go
package models

import (
	"github.com/golang-jwt/jwt/v4"
)

type User struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	Password  string `json:"-"`
	CreatedAt string `json:"created_at"`
}

type Account struct {
	ID          int     `json:"id"`
	UserID      int     `json:"user_id"`
	Balance     float64 `json:"balance"`
	BalanceHash string  `json:"balance_hash"`
	CreatedAt   string  `json:"created_at"`
}

type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// TransactionRequest represents a request to deposit or withdraw funds.
type TransactionRequest struct {
	AccountID     int     `json:"account_id"` // Account to deposit into or withdraw from.
	Amount        float64 `json:"amount"`
	TransactionID string  `json:"transaction_id"` // Added for tracking and idempotency
}

// TransferRequest represents a request to transfer funds between accounts.
type TransferRequest struct {
	FromID int     `json:"from_id"`
	ToID   int     `json:"to_id"`
	Amount float64 `json:"amount"`
}

// Claims represents the JWT claims.
type Claims struct {
	UserID int `json:"user_id"`
	jwt.RegisteredClaims
}
