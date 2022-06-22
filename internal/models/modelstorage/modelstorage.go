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
	OrderNumber int32   `db:"amount"`
	Amount      float64 `db:"amount"`
	ProcessedAt string  `db:"processed_at"`
}
