// Package modelstorage provides types for querying relational DB.

package modelstorage

type UserStorageEntry struct {
	ID           uint   `db:"id"`
	UserID       string `db:"user_id"`
	Login        string `db:"login"`
	Password     string `db:"password"`
	RegisteredAt string `db:"registered_at"`
}

type BalanceStorageEntry struct {
	ID     uint    `db:"id"`
	UserID string  `db:"user_id"`
	Amount float64 `db:"amount"`
}

type WithdrawalStorageEntry struct {
	ID          uint    `db:"id"`
	UserID      string  `db:"user_id"`
	OrderNumber int     `db:"order_number"`
	Amount      float64 `db:"amount"`
	ProcessedAt string  `db:"processed_at"`
}

type OrderStorageEntry struct {
	ID          uint    `db:"id"`
	UserID      string  `db:"user_id"`
	OrderNumber int     `db:"order_number"`
	Status      string  `db:"status"`
	Accrual     float64 `db:"accrual"`
	CreatedAt   string  `db:"created_at"`
}
