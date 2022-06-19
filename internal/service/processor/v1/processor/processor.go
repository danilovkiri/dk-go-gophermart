package processor

import (
	"context"
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
	err := proc.storage.AddNewUser(ctx, credentials, userID)
	if err != nil {
		return nil, err
	}
	return newCookie, nil
}
