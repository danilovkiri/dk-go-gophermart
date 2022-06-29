// Package processor provides intermediary layer functionality between the DB and API endpoint handlers.

package processor

import (
	"context"
	"fmt"
	"github.com/ShiraazMoollatjie/goluhn"
	"github.com/danilovkiri/dk-go-gophermart/internal/models/modeldto"
	"github.com/danilovkiri/dk-go-gophermart/internal/models/modelqueue"
	serviceErrors "github.com/danilovkiri/dk-go-gophermart/internal/service/processor/v1/errors"
	"github.com/danilovkiri/dk-go-gophermart/internal/service/secretary/v1"
	"github.com/danilovkiri/dk-go-gophermart/internal/storage/v1"
	"net/http"
	"sort"
	"strconv"
	"time"
)

// Processor defines attributes of a struct available to its methods.
type Processor struct {
	storage   storage.Storage
	secretary secretary.Secretary
}

// InitService initializes an intermediary service for data processing.
func InitService(st storage.Storage, sec secretary.Secretary) (*Processor, error) {
	if st == nil {
		return nil, &serviceErrors.ServiceFoundNilArgument{Msg: "nil storage was passed to service initializer"}
	}
	if sec == nil {
		return nil, &serviceErrors.ServiceFoundNilArgument{Msg: "nil secretary was passed to service initializer"}
	}
	processor := &Processor{
		storage:   st,
		secretary: sec,
	}
	return processor, nil
}

// AddNewUser processes user register requests.
func (proc *Processor) AddNewUser(ctx context.Context, credentials modeldto.User) (*http.Cookie, error) {
	newCookie, userID := proc.secretary.NewCookie()
	cipheredCredentials := modeldto.User{
		Login:    proc.secretary.Encode(credentials.Login),
		Password: proc.secretary.Encode(credentials.Password),
	}
	err := proc.storage.AddNewUser(ctx, cipheredCredentials, userID)
	if err != nil {
		return nil, err
	}
	return newCookie, nil
}

// LoginUser processes user login requests.
func (proc *Processor) LoginUser(ctx context.Context, credentials modeldto.User) (*http.Cookie, error) {
	cipheredCredentials := modeldto.User{
		Login:    proc.secretary.Encode(credentials.Login),
		Password: proc.secretary.Encode(credentials.Password),
	}
	userID, err := proc.storage.CheckUser(ctx, cipheredCredentials)
	if err != nil {
		return nil, err
	}
	userCookie := proc.secretary.GetCookieForUser(userID)
	return userCookie, nil
}

// GetBalance processes balance query requests.
func (proc *Processor) GetBalance(ctx context.Context, cipheredUserID string) (*modeldto.Balance, error) {
	userID, err := proc.secretary.Decode(cipheredUserID)
	if err != nil {
		return nil, err
	}
	currentAmount, err := proc.storage.GetCurrentAmount(ctx, userID)
	if err != nil {
		return nil, err
	}
	withdrawnAmount, err := proc.storage.GetWithdrawnAmount(ctx, userID)
	if err != nil {
		return nil, err
	}
	balance := modeldto.Balance{
		CurrentAmount:   currentAmount,
		WithdrawnAmount: withdrawnAmount,
	}
	return &balance, nil
}

// GetWithdrawals processes withdrawals query requests.
func (proc *Processor) GetWithdrawals(ctx context.Context, cipheredUserID string) ([]modeldto.Withdrawal, error) {
	userID, err := proc.secretary.Decode(cipheredUserID)
	if err != nil {
		return nil, err
	}
	withdrawals, err := proc.storage.GetWithdrawals(ctx, userID)
	if err != nil {
		return nil, err
	}
	var responseWithdrawals []modeldto.Withdrawal
	for _, withdrawal := range withdrawals {
		responseWithdrawal := modeldto.Withdrawal{
			OrderNumber:     strconv.Itoa(withdrawal.OrderNumber),
			WithdrawnAmount: withdrawal.Amount,
			ProcessedAt:     withdrawal.ProcessedAt,
		}
		responseWithdrawals = append(responseWithdrawals, responseWithdrawal)
	}
	sort.Slice(responseWithdrawals, func(i, j int) bool {
		time1, _ := time.Parse(time.RFC3339, responseWithdrawals[i].ProcessedAt)
		time2, _ := time.Parse(time.RFC3339, responseWithdrawals[j].ProcessedAt)
		return time1.Before(time2)
	})
	return responseWithdrawals, nil
}

// GetOrders processes orders query requests.
func (proc *Processor) GetOrders(ctx context.Context, cipheredUserID string) ([]modeldto.Order, error) {
	userID, err := proc.secretary.Decode(cipheredUserID)
	if err != nil {
		return nil, err
	}
	orders, err := proc.storage.GetOrders(ctx, userID)
	if err != nil {
		return nil, err
	}
	var responseOrders []modeldto.Order
	for _, order := range orders {
		responseOrder := modeldto.Order{
			OrderNumber: strconv.Itoa(order.OrderNumber),
			Status:      order.Status,
			Accrual:     order.Accrual,
			UploadedAt:  order.CreatedAt,
		}
		responseOrders = append(responseOrders, responseOrder)
	}
	sort.Slice(responseOrders, func(i, j int) bool {
		time1, _ := time.Parse(time.RFC3339, responseOrders[i].UploadedAt)
		time2, _ := time.Parse(time.RFC3339, responseOrders[j].UploadedAt)
		return time1.Before(time2)
	})
	return responseOrders, nil
}

// AddNewWithdrawal processes new withdrawal requests.
func (proc *Processor) AddNewWithdrawal(ctx context.Context, cipheredUserID string, withdrawal modeldto.NewOrderWithdrawal) error {
	userID, err := proc.secretary.Decode(cipheredUserID)
	if err != nil {
		return err
	}
	err = goluhn.Validate(withdrawal.OrderNumber)
	if err != nil {
		return &serviceErrors.ServiceIllegalOrderNumber{Msg: fmt.Sprintf("illegal order number %s", withdrawal.OrderNumber)}
	}
	currentAmount, err := proc.storage.GetCurrentAmount(ctx, userID)
	if err != nil {
		return err
	}
	if currentAmount < withdrawal.Amount {
		return &serviceErrors.ServiceNotEnoughFunds{Msg: fmt.Sprintf("not enough funds are available, present - %v, required - %v", currentAmount, withdrawal.Amount)}
	}
	err = proc.storage.AddNewWithdrawal(ctx, userID, withdrawal)
	if err != nil {
		return err
	}
	return nil
}

// AddNewOrder processes new order requests.
func (proc *Processor) AddNewOrder(ctx context.Context, cipheredUserID, orderNumber string) error {
	userID, err := proc.secretary.Decode(cipheredUserID)
	if err != nil {
		return err
	}
	err = goluhn.Validate(orderNumber)
	if err != nil {
		return &serviceErrors.ServiceIllegalOrderNumber{Msg: fmt.Sprintf("illegal order number %s", orderNumber)}
	}
	orderNumberInt, err := strconv.Atoi(orderNumber)
	if err != nil {
		return &serviceErrors.ServiceIllegalOrderNumber{Msg: fmt.Sprintf("illegal order number %s", orderNumber)}
	}
	err = proc.storage.AddNewOrder(ctx, userID, orderNumberInt)
	if err != nil {
		return err
	}
	proc.storage.SendToQueue(modelqueue.OrderQueueEntry{
		UserID:      userID,
		OrderNumber: orderNumberInt,
		OrderStatus: "NEW",
	})
	return nil
}
