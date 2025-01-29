// Path: internal/services/account_service.go
package services

import (
	"bank-api/internal/models"
	"bank-api/pkg/utils"
	"fmt"
	"gorm.io/gorm"
)

// AccountService handles account-related operations.
type AccountService interface {
	GetAccounts(userID uint) ([]models.Account, error)
}

type accountService struct {
	db        *gorm.DB
	secretKey string
}

// NewAccountService creates a new AccountService.
func NewAccountService(db *gorm.DB, secretKey string) AccountService {
	return &accountService{
		db:        db,
		secretKey: secretKey,
	}
}

// GetAccounts retrieves all accounts for a given user.
func (s *accountService) GetAccounts(userID uint) ([]models.Account, error) {
	var accounts []models.Account
	if err := s.db.Where("user_id = ?", userID).Find(&accounts).Error; err != nil {
		return nil, &AppError{Code: 500, Message: "Failed to query accounts", Details: err.Error(), Err: err}
	}

	for _, acc := range accounts {
		// Verify balance integrity
		expectedHash := utils.CreateHMAC(fmt.Sprintf("%f:%d", acc.Balance, acc.ID), []byte(s.secretKey))
		if acc.BalanceHash != expectedHash {
			return nil, &AppError{Code: 500, Message: "Balance integrity check failed", Details: fmt.Sprintf("account_id: %d", acc.ID)}
		}
	}

	return accounts, nil
}
