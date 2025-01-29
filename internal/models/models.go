// Path: internal/models/models.go
package models

import (
	"github.com/golang-jwt/jwt/v4"
)

// Структура Базы данных

// здесь хранится пароль и имя и ID
type User struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	Password  string `json:"-"`
	CreatedAt string `json:"created_at"`
}

// При создании аккаунта получается это тут хранится баланс и ID
type Account struct {
	ID          int     `json:"id"`
	UserID      int     `json:"user_id"`
	Balance     float64 `json:"balance"`
	BalanceHash string  `json:"-"` // Исключено из JSON
	CreatedAt   string  `json:"created_at"`
}

// Записывает в базу данных все входы в аккаунт
type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// // Записывает в базу даннывх все транзакции, пополнения, выводы
type TransactionRequest struct {
	AccountID     int     `json:"account_id"`
	Amount        float64 `json:"amount"`
	TransactionID string  `json:"transaction_id"` // Надо сделать что-бы еще этот ID возвращался во время реквеста, это чтоб админы нашли его.
}

// Записывает в базу данных переводы между счетами
type TransferRequest struct {
	FromID int     `json:"from_id"`
	ToID   int     `json:"to_id"`
	Amount float64 `json:"amount"`
}

// JWT Проверка
type Claims struct {
	UserID int `json:"user_id"`
	jwt.RegisteredClaims
}
