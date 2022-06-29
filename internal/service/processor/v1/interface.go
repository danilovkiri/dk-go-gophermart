// Package processor provides intermediary layer functionality between the DB and API endpoint handlers.

package processor

import (
	"context"
	"github.com/danilovkiri/dk-go-gophermart/internal/models/modeldto"
	"net/http"
)

// Processor defines a set of methods for types implementing Processor.
type Processor interface {
	AddNewUser(ctx context.Context, credentials modeldto.User) (*http.Cookie, error)
	LoginUser(ctx context.Context, credentials modeldto.User) (*http.Cookie, error)
	GetBalance(ctx context.Context, cipheredUserID string) (*modeldto.Balance, error)
	GetWithdrawals(ctx context.Context, cipheredUserID string) ([]modeldto.Withdrawal, error)
	GetOrders(ctx context.Context, cipheredUserID string) ([]modeldto.Order, error)
	AddNewWithdrawal(ctx context.Context, cipheredUserID string, withdrawal modeldto.NewOrderWithdrawal) error
	AddNewOrder(ctx context.Context, cipheredUserID string, orderNumber string) error
}
