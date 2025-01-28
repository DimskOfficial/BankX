// Path: internal/services/account_service.go
package services

import (
	"bank-api/internal/models"
	"bank-api/pkg/utils"
	"database/sql"
	"fmt"
)

// AccountService handles account-related operations.
type AccountService interface {
	GetAccounts(userID int) ([]models.Account, error)
}

type accountService struct {
	db        *sql.DB
	secretKey string
}

// NewAccountService creates a new AccountService.
func NewAccountService(db *sql.DB, secretKey string) AccountService {
	return &accountService{
		db:        db,
		secretKey: secretKey,
	}
}

// GetAccounts retrieves all accounts for a given user.
func (s *accountService) GetAccounts(userID int) ([]models.Account, error) {
	query := `
        SELECT id, user_id, balance, balance_hash, created_at
        FROM accounts
        WHERE user_id = ?
    `
	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, &AppError{Code: 500, Message: "Failed to query accounts", Details: err.Error(), Err: err}
	}
	defer rows.Close()

	var accounts []models.Account
	for rows.Next() {
		var acc models.Account
		if err := rows.Scan(&acc.ID, &acc.UserID, &acc.Balance, &acc.BalanceHash, &acc.CreatedAt); err != nil {
			return nil, &AppError{Code: 500, Message: "Failed to scan account row", Details: err.Error(), Err: err}
		}

		// Verify balance integrity
		expectedHash := utils.CreateHMAC(fmt.Sprintf("%f:%d", acc.Balance, acc.ID), []byte(s.secretKey))
		if acc.BalanceHash != expectedHash {
			return nil, &AppError{Code: 500, Message: "Balance integrity check failed", Details: fmt.Sprintf("account_id: %d", acc.ID)}
		}

		accounts = append(accounts, acc)
	}

	if err = rows.Err(); err != nil {
		return nil, &AppError{Code: 500, Message: "Error iterating account rows", Details: err.Error(), Err: err}
	}

	return accounts, nil
}
