// Package storage provides functionality for operating a relational DB.

package storage

import (
	"context"
	"github.com/danilovkiri/dk-go-gophermart/internal/storage/v1/modelstorage"

	"github.com/danilovkiri/dk-go-gophermart/internal/models/modeldto"
	"github.com/danilovkiri/dk-go-gophermart/internal/models/modelqueue"
)

// RegisterLogin defines a set of methods for types implementing RegisterLogin.
type RegisterLogin interface {
	AddNewUser(ctx context.Context, credentials modeldto.User, userID string) error
	CheckUser(ctx context.Context, credentials modeldto.User) (string, error)
}

// CheckBalance defines a set of methods for types implementing CheckBalance.
type CheckBalance interface {
	GetCurrentAmount(ctx context.Context, userID string) (float64, error)
	GetWithdrawnAmount(ctx context.Context, userID string) (float64, error)
}

// CheckWithdrawals defines a set of methods for types implementing CheckWithdrawals.
type CheckWithdrawals interface {
	GetWithdrawals(ctx context.Context, userID string) ([]modelstorage.WithdrawalStorageEntry, error)
}

// CheckOrders defines a set of methods for types implementing CheckOrders.
type CheckOrders interface {
	GetOrders(ctx context.Context, userID string) ([]modelstorage.OrderStorageEntry, error)
}

// NewWithdrawal defines a set of methods for types implementing NewWithdrawal.
type NewWithdrawal interface {
	AddNewWithdrawal(ctx context.Context, userID string, withdrawal modeldto.NewOrderWithdrawal) error
}

// NewOrder defines a set of methods for types implementing NewOrder.
type NewOrder interface {
	AddNewOrder(ctx context.Context, userID string, orderNumber int) error
	SendToQueue(item modelqueue.OrderQueueEntry)
}

// Storage defines a set of methods for types implementing Storage.
type Storage interface {
	RegisterLogin
	CheckBalance
	CheckWithdrawals
	CheckOrders
	NewWithdrawal
	NewOrder
}
