// Path: internal/services/transaction_service.go
package services

import (
	"bank-api/internal/models"
	"bank-api/pkg/utils"
	"database/sql"
	"errors"
	"fmt"
)

// TransactionService handles transaction-related operations.
type TransactionService interface {
	ProcessDeposit(req *models.TransactionRequest, claims *models.Claims) error
	ProcessWithdraw(req *models.TransactionRequest, claims *models.Claims) error
	ProcessTransfer(req *models.TransferRequest, claims *models.Claims) error
}

type transactionService struct {
	db        *sql.DB
	secretKey string
}

// NewTransactionService creates a new TransactionService.
func NewTransactionService(db *sql.DB, secretKey string) TransactionService {
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

	tx, err := s.db.Begin()
	if err != nil {
		return &AppError{Code: 500, Message: "Failed to start transaction", Details: err.Error(), Err: err}
	}
	defer tx.Rollback()

	// Check if the account exists and belongs to the user.
	var (
		existingBalance float64
		balanceHash     string
	)

	err = tx.QueryRow(`SELECT balance, balance_hash FROM accounts WHERE id = ? AND user_id = ?`, req.AccountID, claims.UserID).Scan(&existingBalance, &balanceHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &AppError{Code: 404, Message: "Account not found or access denied", Details: fmt.Sprintf("account_id: %d, user_id: %d", req.AccountID, claims.UserID)}
		}
		return &AppError{Code: 500, Message: "Failed to query account", Details: err.Error(), Err: err}
	}

	// Verify balance hash
	expectedHash := utils.CreateHMAC(fmt.Sprintf("%f:%d", existingBalance, req.AccountID), []byte(s.secretKey))
	if balanceHash != expectedHash {
		return &AppError{Code: 500, Message: "Balance integrity check failed", Details: fmt.Sprintf("account_id: %d", req.AccountID)}
	}

	// Update the account balance and hash.
	newBalance := existingBalance + req.Amount
	newBalanceHash := utils.CreateHMAC(fmt.Sprintf("%f:%d", newBalance, req.AccountID), []byte(s.secretKey))
	_, err = tx.Exec(`UPDATE accounts SET balance = ?, balance_hash = ? WHERE id = ?`, newBalance, newBalanceHash, req.AccountID)
	if err != nil {
		return &AppError{Code: 500, Message: "Failed to update account balance", Details: err.Error(), Err: err}
	}
	req.TransactionID = utils.GenerateTransactionID() // Генерация transactionID

	// Insert the transaction record.
	_, err = tx.Exec(`
        INSERT INTO transactions (id, to_account_id, amount, type, status, created_at)
        VALUES (?, ?, ?, ?, ?, ?)`,
		req.TransactionID, req.AccountID, req.Amount, "deposit", "completed", utils.GetCurrentTimestamp())
	if err != nil {
		return &AppError{Code: 500, Message: "Failed to insert transaction record", Details: err.Error(), Err: err}
	}

	if err := tx.Commit(); err != nil {
		return &AppError{Code: 500, Message: "Failed to commit transaction", Details: err.Error(), Err: err}
	}
	return nil
}

// ProcessWithdraw handles a withdrawal transaction.
func (s *transactionService) ProcessWithdraw(req *models.TransactionRequest, claims *models.Claims) error {
	if req.Amount <= 0 {
		return &AppError{Code: 400, Message: "Invalid withdrawal amount", Details: "Amount must be positive"}
	}

	tx, err := s.db.Begin()
	if err != nil {
		return &AppError{Code: 500, Message: "Failed to start transaction", Details: err.Error(), Err: err}
	}
	defer tx.Rollback()

	// Check if the account exists, belongs to the user, and has sufficient funds.
	var (
		existingBalance float64
		balanceHash     string
	)
	err = tx.QueryRow(`SELECT balance, balance_hash FROM accounts WHERE id = ? AND user_id = ?`, req.AccountID, claims.UserID).Scan(&existingBalance, &balanceHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &AppError{Code: 404, Message: "Account not found or access denied", Details: fmt.Sprintf("account_id: %d, user_id: %d", req.AccountID, claims.UserID)}
		}
		return &AppError{Code: 500, Message: "Failed to query account", Details: err.Error(), Err: err}
	}

	// Verify balance hash
	expectedHash := utils.CreateHMAC(fmt.Sprintf("%f:%d", existingBalance, req.AccountID), []byte(s.secretKey))
	if balanceHash != expectedHash {
		return &AppError{Code: 500, Message: "Balance integrity check failed", Details: fmt.Sprintf("account_id: %d", req.AccountID)}
	}

	if existingBalance < req.Amount {
		return &AppError{Code: 400, Message: "Insufficient funds", Details: fmt.Sprintf("account_id: %d, balance: %f, requested: %f", req.AccountID, existingBalance, req.Amount)}
	}

	// Update account balance and hash.
	newBalance := existingBalance - req.Amount
	newBalanceHash := utils.CreateHMAC(fmt.Sprintf("%f:%d", newBalance, req.AccountID), []byte(s.secretKey))
	_, err = tx.Exec(`UPDATE accounts SET balance = ?, balance_hash = ? WHERE id = ? AND user_id = ?`, newBalance, newBalanceHash, req.AccountID, claims.UserID)
	if err != nil {
		return &AppError{Code: 500, Message: "Failed to update account balance", Details: err.Error(), Err: err}
	}
	req.TransactionID = utils.GenerateTransactionID() // Генерация transactionID

	// Insert transaction record.
	_, err = tx.Exec(`
        INSERT INTO transactions (id, from_account_id, amount, type, status, created_at)
        VALUES (?, ?, ?, ?, ?, ?)`,
		req.TransactionID, req.AccountID, req.Amount, "withdraw", "completed", utils.GetCurrentTimestamp())
	if err != nil {
		return &AppError{Code: 500, Message: "Failed to insert transaction record", Details: err.Error(), Err: err}
	}

	if err := tx.Commit(); err != nil {
		return &AppError{Code: 500, Message: "Failed to commit transaction", Details: err.Error(), Err: err}
	}
	return nil
}

// ProcessTransfer handles a fund transfer between two accounts.
// ProcessTransfer handles a fund transfer between two accounts.
func (s *transactionService) ProcessTransfer(req *models.TransferRequest, claims *models.Claims) error {
	if req.Amount <= 0 {
		return &AppError{Code: 400, Message: "Invalid transfer amount", Details: "Amount must be positive"}
	}
	if req.FromID == req.ToID {
		return &AppError{Code: 400, Message: "Invalid transfer", Details: "Source and destination accounts must be different"}
	}

	tx, err := s.db.Begin()
	if err != nil {
		return &AppError{Code: 500, Message: "Failed to start transaction", Details: err.Error(), Err: err}
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	// Check if the source account exists, belongs to the user, and has sufficient funds.
	var (
		fromBalance     float64
		fromBalanceHash string
	)
	err = tx.QueryRow(`SELECT balance, balance_hash FROM accounts WHERE id = ? AND user_id = ?`, req.FromID, claims.UserID).Scan(&fromBalance, &fromBalanceHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &AppError{Code: 404, Message: "Source account not found or access denied", Details: fmt.Sprintf("account_id: %d, user_id: %d", req.FromID, claims.UserID)}
		}
		return &AppError{Code: 500, Message: "Failed to query source account", Details: err.Error(), Err: err}
	}

	// Verify balance hash of the source account.
	expectedFromHash := utils.CreateHMAC(fmt.Sprintf("%f:%d", fromBalance, req.FromID), []byte(s.secretKey))
	if fromBalanceHash != expectedFromHash {
		return &AppError{Code: 500, Message: "Source account balance integrity check failed", Details: fmt.Sprintf("account_id: %d", req.FromID)}
	}

	if fromBalance < req.Amount {
		return &AppError{Code: 400, Message: "Insufficient funds in source account", Details: fmt.Sprintf("account_id: %d, balance: %f, requested: %f", req.FromID, fromBalance, req.Amount)}
	}

	// Check if the destination account exists.
	var (
		toBalance     float64
		toBalanceHash string
	)

	err = tx.QueryRow(`SELECT balance, balance_hash FROM accounts WHERE id = ?`, req.ToID).Scan(&toBalance, &toBalanceHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &AppError{Code: 404, Message: "Destination account not found", Details: fmt.Sprintf("account_id: %d", req.ToID)}
		}
		return &AppError{Code: 500, Message: "Failed to query destination account", Details: err.Error(), Err: err}
	}

	// Verify balance hash of the destination account
	expectedToHash := utils.CreateHMAC(fmt.Sprintf("%f:%d", toBalance, req.ToID), []byte(s.secretKey))
	if toBalanceHash != expectedToHash {
		return &AppError{Code: 500, Message: "Destination account balance integrity check failed", Details: fmt.Sprintf("account_id: %d", req.ToID)}
	}

	// Perform the transfer (update balances and hashes).
	newFromBalance := fromBalance - req.Amount
	newFromBalanceHash := utils.CreateHMAC(fmt.Sprintf("%f:%d", newFromBalance, req.FromID), []byte(s.secretKey))
	_, err = tx.Exec(`UPDATE accounts SET balance = ?, balance_hash = ? WHERE id = ? AND user_id = ?`, newFromBalance, newFromBalanceHash, req.FromID, claims.UserID)
	if err != nil {
		return &AppError{Code: 500, Message: "Failed to update source account balance", Details: err.Error(), Err: err}
	}

	newToBalance := toBalance + req.Amount
	newToBalanceHash := utils.CreateHMAC(fmt.Sprintf("%f:%d", newToBalance, req.ToID), []byte(s.secretKey))
	_, err = tx.Exec(`UPDATE accounts SET balance = ?, balance_hash = ? WHERE id = ?`, newToBalance, newToBalanceHash, req.ToID)
	if err != nil {
		return &AppError{Code: 500, Message: "Failed to update destination account balance", Details: err.Error(), Err: err}
	}

	transactionID := utils.GenerateTransactionID()
	// Кароче успешная транзакция.
	_, err = tx.Exec(`
        INSERT INTO transactions (id, from_account_id, to_account_id, amount, type, status, created_at)
        VALUES (?, ?, ?, ?, ?, ?, ?)`,
		transactionID, req.FromID, req.ToID, req.Amount, "transfer", "completed", utils.GetCurrentTimestamp())
	if err != nil {
		return &AppError{Code: 500, Message: "Failed to insert transaction record", Details: err.Error(), Err: err}
	}

	if err := tx.Commit(); err != nil {
		return &AppError{Code: 500, Message: "Failed to commit transaction", Details: err.Error(), Err: err}
	}
	return nil
}
