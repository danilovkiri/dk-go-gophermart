package storage

import (
	"context"
	"github.com/danilovkiri/dk-go-gophermart/internal/models/modeluser"
)

type Register interface {
	AddNewUser(ctx context.Context, credentials modeluser.ModelCredentials, userID string) (err error)
}

type Storage interface {
	Register
}
