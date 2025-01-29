// Path: internal/services/transaction_service.go
package services

import (
	"bank-api/internal/models"
	"bank-api/pkg/utils"
	"errors"
	"fmt"
	"gorm.io/gorm"
)

// TransactionService handles transaction-related operations.
type TransactionService interface {
	ProcessDeposit(req *models.TransactionRequest, claims *models.Claims) error
	ProcessWithdraw(req *models.TransactionRequest, claims *models.Claims) error
	ProcessTransfer(req *models.TransferRequest, claims *models.Claims) error
}

type transactionService struct {
	db        *gorm.DB
	secretKey string
}

// NewTransactionService creates a new TransactionService.
func NewTransactionService(db *gorm.DB, secretKey string) TransactionService {
	return &transactionService{
		db:        db,
		secretKey: secretKey,
	}
}

// AppError is a custom error type that includes an HTTP status code.
type AppError struct {
	Code    int    `json:"-"`
	Message string `json:"message"`
	Details string `json:"details"`
	Err     error  `json:"-"`
}

func (e *AppError) Error() string {
	return fmt.Sprintf("AppError: %s (Code: %d, Details: %s)", e.Message, e.Code, e.Details)
}

// ProcessDeposit handles a deposit transaction.
func (s *transactionService) ProcessDeposit(req *models.TransactionRequest, claims *models.Claims) error {
	if req.Amount <= 0 {
		return &AppError{Code: 400, Message: "Invalid deposit amount", Details: "Amount must be positive"}
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		var account models.Account
		if err := tx.Where("id = ? AND user_id = ?", req.AccountID, claims.UserID).First(&account).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return &AppError{Code: 404, Message: "Account not found or access denied", Details: fmt.Sprintf("account_id: %d, user_id: %d", req.AccountID, claims.UserID)}
			}
			return &AppError{Code: 500, Message: "Failed to query account", Details: err.Error(), Err: err}
		}

		// Verify balance hash
		expectedHash := utils.CreateHMAC(fmt.Sprintf("%f:%d", account.Balance, req.AccountID), []byte(s.secretKey))
		if account.BalanceHash != expectedHash {
			return &AppError{Code: 500, Message: "Balance integrity check failed", Details: fmt.Sprintf("account_id: %d", req.AccountID)}
		}

		// Update the account balance and hash.
		account.Balance += req.Amount
		account.BalanceHash = utils.CreateHMAC(fmt.Sprintf("%f:%d", account.Balance, req.AccountID), []byte(s.secretKey))
		if err := tx.Save(&account).Error; err != nil {
			return &AppError{Code: 500, Message: "Failed to update account balance", Details: err.Error(), Err: err}
		}

		req.TransactionID = utils.GenerateTransactionID() // Генерация transactionID

		// Insert the transaction record.
		transaction := models.Transaction{
			ID:          req.TransactionID,
			ToAccountID: &req.AccountID,
			Amount:      req.Amount,
			Type:        "deposit",
			Status:      "completed",
			CreatedAt:   utils.GetCurrentTimestamp(),
		}
		if err := tx.Create(&transaction).Error; err != nil {
			return &AppError{Code: 500, Message: "Failed to insert transaction record", Details: err.Error(), Err: err}
		}

		return nil
	})
}

// ProcessWithdraw handles a withdrawal transaction.
func (s *transactionService) ProcessWithdraw(req *models.TransactionRequest, claims *models.Claims) error {
	if req.Amount <= 0 {
		return &AppError{Code: 400, Message: "Invalid withdrawal amount", Details: "Amount must be positive"}
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		var account models.Account
		if err := tx.Where("id = ? AND user_id = ?", req.AccountID, claims.UserID).First(&account).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return &AppError{Code: 404, Message: "Account not found or access denied", Details: fmt.Sprintf("account_id: %d, user_id: %d", req.AccountID, claims.UserID)}
			}
			return &AppError{Code: 500, Message: "Failed to query account", Details: err.Error(), Err: err}
		}

		// Verify balance hash
		expectedHash := utils.CreateHMAC(fmt.Sprintf("%f:%d", account.Balance, req.AccountID), []byte(s.secretKey))
		if account.BalanceHash != expectedHash {
			return &AppError{Code: 500, Message: "Balance integrity check failed", Details: fmt.Sprintf("account_id: %d", req.AccountID)}
		}

		if account.Balance < req.Amount {
			return &AppError{Code: 400, Message: "Insufficient funds", Details: fmt.Sprintf("account_id: %d, balance: %f, requested: %f", req.AccountID, account.Balance, req.Amount)}
		}

		// Update account balance and hash.
		account.Balance -= req.Amount
		account.BalanceHash = utils.CreateHMAC(fmt.Sprintf("%f:%d", account.Balance, req.AccountID), []byte(s.secretKey))
		if err := tx.Save(&account).Error; err != nil {
			return &AppError{Code: 500, Message: "Failed to update account balance", Details: err.Error(), Err: err}
		}

		req.TransactionID = utils.GenerateTransactionID() // Генерация transactionID

		// Insert transaction record.
		transaction := models.Transaction{
			ID:            req.TransactionID,
			FromAccountID: &req.AccountID,
			Amount:        req.Amount,
			Type:          "withdraw",
			Status:        "completed",
			CreatedAt:     utils.GetCurrentTimestamp(),
		}
		if err := tx.Create(&transaction).Error; err != nil {
			return &AppError{Code: 500, Message: "Failed to insert transaction record", Details: err.Error(), Err: err}
		}

		return nil
	})
}

// ProcessTransfer handles a fund transfer between two accounts.
func (s *transactionService) ProcessTransfer(req *models.TransferRequest, claims *models.Claims) error {
	if req.Amount <= 0 {
		return &AppError{Code: 400, Message: "Invalid transfer amount", Details: "Amount must be positive"}
	}
	if req.FromID == req.ToID {
		return &AppError{Code: 400, Message: "Invalid transfer", Details: "Source and destination accounts must be different"}
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		var fromAccount, toAccount models.Account

		// Check if the source account exists, belongs to the user, and has sufficient funds.
		if err := tx.Where("id = ? AND user_id = ?", req.FromID, claims.UserID).First(&fromAccount).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return &AppError{Code: 404, Message: "Source account not found or access denied", Details: fmt.Sprintf("account_id: %d, user_id: %d", req.FromID, claims.UserID)}
			}
			return &AppError{Code: 500, Message: "Failed to query source account", Details: err.Error(), Err: err}
		}

		// Verify balance hash of the source account.
		expectedFromHash := utils.CreateHMAC(fmt.Sprintf("%f:%d", fromAccount.Balance, req.FromID), []byte(s.secretKey))
		if fromAccount.BalanceHash != expectedFromHash {
			return &AppError{Code: 500, Message: "Source account balance integrity check failed", Details: fmt.Sprintf("account_id: %d", req.FromID)}
		}

		if fromAccount.Balance < req.Amount {
			return &AppError{Code: 400, Message: "Insufficient funds in source account", Details: fmt.Sprintf("account_id: %d, balance: %f, requested: %f", req.FromID, fromAccount.Balance, req.Amount)}
		}

		// Check if the destination account exists.
		if err := tx.Where("id = ?", req.ToID).First(&toAccount).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return &AppError{Code: 404, Message: "Destination account not found", Details: fmt.Sprintf("account_id: %d", req.ToID)}
			}
			return &AppError{Code: 500, Message: "Failed to query destination account", Details: err.Error(), Err: err}
		}

		// Verify balance hash of the destination account
		expectedToHash := utils.CreateHMAC(fmt.Sprintf("%f:%d", toAccount.Balance, req.ToID), []byte(s.secretKey))
		if toAccount.BalanceHash != expectedToHash {
			return &AppError{Code: 500, Message: "Destination account balance integrity check failed", Details: fmt.Sprintf("account_id: %d", req.ToID)}
		}

		// Perform the transfer (update balances and hashes).
		fromAccount.Balance -= req.Amount
		fromAccount.BalanceHash = utils.CreateHMAC(fmt.Sprintf("%f:%d", fromAccount.Balance, req.FromID), []byte(s.secretKey))
		if err := tx.Save(&fromAccount).Error; err != nil {
			return &AppError{Code: 500, Message: "Failed to update source account balance", Details: err.Error(), Err: err}
		}

		toAccount.Balance += req.Amount
		toAccount.BalanceHash = utils.CreateHMAC(fmt.Sprintf("%f:%d", toAccount.Balance, req.ToID), []byte(s.secretKey))
		if err := tx.Save(&toAccount).Error; err != nil {
			return &AppError{Code: 500, Message: "Failed to update destination account balance", Details: err.Error(), Err: err}
		}

		transactionID := utils.GenerateTransactionID()
		// Кароче успешная транзакция.
		transaction := models.Transaction{
			ID:            transactionID,
			FromAccountID: &req.FromID,
			ToAccountID:   &req.ToID,
			Amount:        req.Amount,
			Type:          "transfer",
			Status:        "completed",
			CreatedAt:     utils.GetCurrentTimestamp(),
		}
		if err := tx.Create(&transaction).Error; err != nil {
			return &AppError{Code: 500, Message: "Failed to insert transaction record", Details: err.Error(), Err: err}
		}

		return nil
	})
}
