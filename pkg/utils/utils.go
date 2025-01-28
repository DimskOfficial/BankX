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

// GenerateRandomString generates a random string of the given length.
func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

// CreateHMAC creates an HMAC-SHA256 hash of the given data.
func CreateHMAC(data string, secret []byte) string {
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// GenerateTransactionID generates a unique transaction ID.
func GenerateTransactionID() string {
	timestamp := time.Now().UnixNano()
	random := GenerateRandomString(8)
	return fmt.Sprintf("%d-%s", timestamp, random)
}

// GetCurrentTimestamp returns the current timestamp in RFC3339 format.
func GetCurrentTimestamp() string {
	return time.Now().Format(time.RFC3339)
}

func CalculateBalanceHash(balance float64, accountID int, secretKey string) string {
	h := hmac.New(sha256.New, []byte(secretKey))
	data := fmt.Sprintf("%f:%d", balance, accountID)
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}
