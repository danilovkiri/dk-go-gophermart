package storage

import (
	"context"
	"github.com/danilovkiri/dk-go-gophermart/internal/models/modelstorage"
	"github.com/danilovkiri/dk-go-gophermart/internal/models/modeluser"
)

type RegisterLogin interface {
	AddNewUser(ctx context.Context, credentials modeluser.ModelCredentials, userID string) error
	CheckUser(ctx context.Context, credentials modeluser.ModelCredentials) (string, error)
}

type CheckBalance interface {
	GetCurrentAmount(ctx context.Context, userID string) (float64, error)
	GetWithdrawnAmount(ctx context.Context, userID string) (float64, error)
}

type CheckWithdrawals interface {
	GetWithdrawals(ctx context.Context, userID string) ([]modelstorage.WithdrawalStorageEntry, error)
}

type Storage interface {
	RegisterLogin
	CheckBalance
	CheckWithdrawals
}
