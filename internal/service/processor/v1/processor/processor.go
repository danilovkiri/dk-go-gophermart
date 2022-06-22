package processor

import (
	"context"
	"github.com/danilovkiri/dk-go-gophermart/internal/models/modeldto"
	"github.com/danilovkiri/dk-go-gophermart/internal/models/modeluser"
	serviceErrors "github.com/danilovkiri/dk-go-gophermart/internal/service/processor/v1/errors"
	"github.com/danilovkiri/dk-go-gophermart/internal/service/secretary/v1"
	"github.com/danilovkiri/dk-go-gophermart/internal/storage/v1"
	"net/http"
)

type Processor struct {
	storage   storage.Storage
	secretary secretary.Secretary
}

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

func (proc *Processor) AddNewUser(ctx context.Context, credentials modeluser.ModelCredentials) (*http.Cookie, error) {
	newCookie, userID := proc.secretary.NewCookie()
	cipheredCredentials := modeluser.ModelCredentials{
		Login:    proc.secretary.Encode(credentials.Login),
		Password: proc.secretary.Encode(credentials.Password),
	}
	err := proc.storage.AddNewUser(ctx, cipheredCredentials, userID)
	if err != nil {
		return nil, err
	}
	return newCookie, nil
}

func (proc *Processor) LoginUser(ctx context.Context, credentials modeluser.ModelCredentials) (*http.Cookie, error) {
	cipheredCredentials := modeluser.ModelCredentials{
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
