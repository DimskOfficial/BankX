// Path: pkg/utils/utils.go
package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"time"
)

// Еебать старье, крч просто функция по рандомному созданию стринга
func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

// CreateHMAC создает хэш HMAC-SHA256 для защиты
func CreateHMAC(data string, secret []byte) string {
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// Генератор транзакционных ID
func GenerateTransactionID() string {
	timestamp := time.Now().UnixNano()
	random := GenerateRandomString(8)
	return fmt.Sprintf("%d-%s", timestamp, random)
}

// GetCurrentTimestamp возвращает текущую временную метку в формате RFC3339.
func GetCurrentTimestamp() string {
	return time.Now().Format(time.RFC3339)
}

func CalculateBalanceHash(balance float64, accountID int, secretKey string) string {
	h := hmac.New(sha256.New, []byte(secretKey))
	data := fmt.Sprintf("%f:%d", balance, accountID)
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}
