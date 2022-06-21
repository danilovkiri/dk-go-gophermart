package modelstorage

type UsersStorageEntry struct {
	ID           uint   `db:"id"`
	UserID       string `db:"user_id"`
	Login        string `db:"login"`
	Password     string `db:"password"`
	RegisteredAt string `db:"registered_at"`
}
