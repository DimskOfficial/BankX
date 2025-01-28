// Path: pkg/database/database.go
package database

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
)

// InitDB initializes the database and creates tables if they don't exist.
func InitDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign key support.
	_, err = db.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	if err := createTables(db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

// createTables creates the necessary database tables.
func createTables(db *sql.DB) error {
	// Use a single transaction for table creation.
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback() // Rollback if any error occurs.

	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
                        id INTEGER PRIMARY KEY AUTOINCREMENT,
                        username TEXT UNIQUE NOT NULL,
                        password TEXT NOT NULL,
                        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
                );`,
		`CREATE TABLE IF NOT EXISTS accounts (
                        id INTEGER PRIMARY KEY AUTOINCREMENT,
                        user_id INTEGER NOT NULL,
                        balance REAL NOT NULL DEFAULT 0,
                        balance_hash TEXT NOT NULL,
                        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
                );`,
		`CREATE TABLE IF NOT EXISTS transactions (
                        id TEXT PRIMARY KEY,
                        from_account_id INTEGER,
                        to_account_id INTEGER,
                        amount REAL NOT NULL,
                        type TEXT NOT NULL,
                        status TEXT NOT NULL,
                        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                        FOREIGN KEY (from_account_id) REFERENCES accounts(id) ON DELETE SET NULL,
                        FOREIGN KEY (to_account_id) REFERENCES accounts(id) ON DELETE SET NULL
                );`,
		// Create indexes for faster lookups.
		`CREATE INDEX IF NOT EXISTS idx_accounts_user_id ON accounts(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_transactions_from_account_id ON transactions(from_account_id);`, // Corrected index name
		`CREATE INDEX IF NOT EXISTS idx_transactions_to_account_id ON transactions(to_account_id);`,     // Corrected index name
	}

	for _, query := range queries {
		_, err = tx.Exec(query)
		if err != nil {
			return fmt.Errorf("failed to execute query: %s, error: %w", query, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
