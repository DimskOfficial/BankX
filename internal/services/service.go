package services

import (
	"bank-api/internal/models"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Login(username, password string) (string, error) {
	var user struct {
		ID       int
		Password string
	}

	err := s.db.QueryRow("SELECT id, password FROM users WHERE username = ?",
		username).Scan(&user.ID, &user.Password)
	if err != nil {
		return "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", errors.New("неверный пароль")
	}

	// Генерация JWT токена
	token := "jwt_token" // Упрощено
	return token, nil
}

func (s *Service) GetAccounts(userID int) ([]models.Account, error) {
	rows, err := s.db.Query("SELECT id, balance FROM accounts WHERE user_id = ?", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []models.Account
	for rows.Next() {
		var acc models.Account
		if err := rows.Scan(&acc.ID, &acc.Balance); err != nil {
			return nil, err
		}
		accounts = append(accounts, acc)
	}

	return accounts, nil
}

func (s *Service) Deposit(userID int, accountID string, amount float64) error {
	_, err := s.db.Exec("UPDATE accounts SET balance = balance + ? WHERE id = ? AND user_id = ?",
		amount, accountID, userID)
	return err
}

func (s *Service) Withdraw(userID int, accountID string, amount float64) error {
	_, err := s.db.Exec("UPDATE accounts SET balance = balance - ? WHERE id = ? AND user_id = ? AND balance >= ?",
		amount, accountID, userID, amount)
	return err
}

func (s *Service) Transfer(userID int, req *models.TransferRequest) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Проверяем владельца счета
	var balance float64
	err = tx.QueryRow("SELECT balance FROM accounts WHERE id = ? AND user_id = ?",
		req.FromID, userID).Scan(&balance)
	if err != nil {
		return err
	}

	if balance < req.Amount {
		return errors.New("недостаточно средств")
	}

	// Выполняем перевод
	_, err = tx.Exec("UPDATE accounts SET balance = balance - ? WHERE id = ?",
		req.Amount, req.FromID)
	if err != nil {
		return err
	}

	_, err = tx.Exec("UPDATE accounts SET balance = balance + ? WHERE id = ?",
		req.Amount, req.ToID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func CalculateBalanceHash(balance float64, accountID int, secretKey string) string {
	h := hmac.New(sha256.New, []byte(secretKey))
	data := fmt.Sprintf("%f:%d", balance, accountID)
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}
