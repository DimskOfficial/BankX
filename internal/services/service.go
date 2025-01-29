package services

import (
	"bank-api/internal/models"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Login(username, password string) (string, error) {
	var user models.User

	err := s.db.Where("username = ?", username).First(&user).Error
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

func (s *Service) GetAccounts(userID uint) ([]models.Account, error) {
	var accounts []models.Account
	err := s.db.Where("user_id = ?", userID).Find(&accounts).Error
	if err != nil {
		return nil, err
	}
	return accounts, nil
}

func (s *Service) Deposit(userID uint, accountID uint, amount float64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var account models.Account
		if err := tx.Where("id = ? AND user_id = ?", accountID, userID).First(&account).Error; err != nil {
			return err
		}

		account.Balance += amount
		return tx.Save(&account).Error
	})
}

func (s *Service) Withdraw(userID uint, accountID uint, amount float64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var account models.Account
		if err := tx.Where("id = ? AND user_id = ?", accountID, userID).First(&account).Error; err != nil {
			return err
		}

		if account.Balance < amount {
			return errors.New("недостаточно средств")
		}

		account.Balance -= amount
		return tx.Save(&account).Error
	})
}

func (s *Service) Transfer(userID uint, req *models.TransferRequest) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var fromAccount, toAccount models.Account

		// Проверяем владельца счета и баланс
		if err := tx.Where("id = ? AND user_id = ?", req.FromID, userID).First(&fromAccount).Error; err != nil {
			return err
		}

		if fromAccount.Balance < req.Amount {
			return errors.New("недостаточно средств")
		}

		// Выполняем перевод
		if err := tx.Where("id = ?", req.ToID).First(&toAccount).Error; err != nil {
			return err
		}

		fromAccount.Balance -= req.Amount
		toAccount.Balance += req.Amount

		if err := tx.Save(&fromAccount).Error; err != nil {
			return err
		}

		if err := tx.Save(&toAccount).Error; err != nil {
			return err
		}

		return nil
	})
}

func CalculateBalanceHash(balance float64, accountID uint, secretKey string) string {
	h := hmac.New(sha256.New, []byte(secretKey))
	data := fmt.Sprintf("%f:%d", balance, accountID)
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}
