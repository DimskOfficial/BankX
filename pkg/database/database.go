// Path: pkg/database/database.go
package database

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// User represents a user in the database.
type User struct {
	ID        uint   `gorm:"primaryKey"`
	Username  string `gorm:"unique;not null"`
	Password  string `gorm:"not null"`
	CreatedAt string `gorm:"not null"`
}

// Account represents an account in the database.
type Account struct {
	ID          uint    `gorm:"primaryKey"`
	UserID      uint    `gorm:"not null"`
	Balance     float64 `gorm:"not null;default:0"`
	BalanceHash string  `gorm:"not null"`
	CreatedAt   string  `gorm:"not null"`
	User        User    `gorm:"constraint:OnDelete:CASCADE;"`
}

// Transaction represents a transaction in the database.
type Transaction struct {
	ID            string `gorm:"primaryKey"`
	FromAccountID *uint
	ToAccountID   *uint
	Amount        float64  `gorm:"not null"`
	Type          string   `gorm:"not null"`
	Status        string   `gorm:"not null"`
	CreatedAt     string   `gorm:"not null"`
	FromAccount   *Account `gorm:"constraint:OnDelete:SET NULL;"`
	ToAccount     *Account `gorm:"constraint:OnDelete:SET NULL;"`
}

// InitDB initializes the database and creates tables if they don't exist.
func InitDB(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := createTables(db); err != nil {
		return nil, err
	}

	return db, nil
}

// createTables creates the necessary tables in the database.
func createTables(db *gorm.DB) error {
	err := db.AutoMigrate(&User{}, &Account{}, &Transaction{})
	if err != nil {
		return fmt.Errorf("failed to auto-migrate tables: %w", err)
	}

	return nil
}
