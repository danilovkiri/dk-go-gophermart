package processor

import (
	"context"
	"github.com/danilovkiri/dk-go-gophermart/internal/models/modeldto"
	"github.com/danilovkiri/dk-go-gophermart/internal/models/modeluser"
	"net/http"
)

type Processor interface {
	AddNewUser(ctx context.Context, credentials modeluser.ModelCredentials) (*http.Cookie, error)
	LoginUser(ctx context.Context, credentials modeluser.ModelCredentials) (*http.Cookie, error)
	GetBalance(ctx context.Context, cipheredUserID string) (*modeldto.Balance, error)
}
