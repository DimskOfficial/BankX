// Path: internal/models/models.go
package models

import (
	"github.com/golang-jwt/jwt/v4"
	"time"
)

// User represents a user in the database.
type User struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	Password  string `json:"-"`
	CreatedAt string `json:"created_at"`
}

// Account represents an account in the database.
type Account struct {
	ID          int     `json:"id"`
	UserID      int     `json:"user_id"`
	Balance     float64 `json:"balance"`
	BalanceHash string  `json:"-"` // Excluded from JSON
	CreatedAt   string  `json:"created_at"`
}

// AuthRequest represents a request for user authentication.
type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// TransactionRequest represents a request for a transaction.
type TransactionRequest struct {
	AccountID     int     `json:"account_id"`
	Amount        float64 `json:"amount"`
	TransactionID string  `json:"transaction_id"` // This should be returned during the request for admin tracking.
}

// TransferRequest represents a request for transferring funds between accounts.
type TransferRequest struct {
	FromID int     `json:"from_id"`
	ToID   int     `json:"to_id"`
	Amount float64 `json:"amount"`
}

// Claims represents JWT claims.
type Claims struct {
	UserID uint `json:"user_id"`
	jwt.RegisteredClaims
}

// Transaction represents a transaction in the database.
type Transaction struct {
	ID            string    `json:"id"`
	FromAccountID *int      `json:"from_account_id"` // Nullable for deposits
	ToAccountID   *int      `json:"to_account_id"`   // Nullable for withdrawals
	Amount        float64   `json:"amount"`
	Type          string    `json:"type"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}
