package storage

import (
	"context"
	"github.com/danilovkiri/dk-go-gophermart/internal/models/modeldto"
	"github.com/danilovkiri/dk-go-gophermart/internal/models/modelstorage"
)

type RegisterLogin interface {
	AddNewUser(ctx context.Context, credentials modeldto.User, userID string) error
	CheckUser(ctx context.Context, credentials modeldto.User) (string, error)
}

type CheckBalance interface {
	GetCurrentAmount(ctx context.Context, userID string) (float64, error)
	GetWithdrawnAmount(ctx context.Context, userID string) (float64, error)
}

type CheckWithdrawals interface {
	GetWithdrawals(ctx context.Context, userID string) ([]modelstorage.WithdrawalStorageEntry, error)
}

type CheckOrders interface {
	GetOrders(ctx context.Context, userID string) ([]modelstorage.OrderStorageEntry, error)
}

type NewWithdrawal interface {
	AddNewWithdrawal(ctx context.Context, userID string, withdrawal modeldto.NewOrderWithdrawal) error
}

type Storage interface {
	RegisterLogin
	CheckBalance
	CheckWithdrawals
	CheckOrders
	NewWithdrawal
}
