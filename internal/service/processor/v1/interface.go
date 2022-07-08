// Package processor provides intermediary layer functionality between the DB and API endpoint handlers.

package processor

import (
	"context"

	"github.com/danilovkiri/dk-go-gophermart/internal/models/modeldto"
)

// Processor defines a set of methods for types implementing Processor.
type Processor interface {
	AddNewUser(ctx context.Context, credentials modeldto.User) (string, error)
	LoginUser(ctx context.Context, credentials modeldto.User) (string, error)
	GetBalance(ctx context.Context, userID string) (*modeldto.Balance, error)
	GetWithdrawals(ctx context.Context, userID string) ([]modeldto.Withdrawal, error)
	GetOrders(ctx context.Context, userID string) ([]modeldto.Order, error)
	AddNewWithdrawal(ctx context.Context, userID string, withdrawal modeldto.NewOrderWithdrawal) error
	AddNewOrder(ctx context.Context, userID string, orderNumber string) error
	GetUserID(accessToken string) (string, error)
}
