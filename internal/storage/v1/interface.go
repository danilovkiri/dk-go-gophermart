package storage

import (
	"context"
	"github.com/danilovkiri/dk-go-gophermart/internal/models/modeluser"
)

type Register interface {
	AddNewUser(ctx context.Context, credentials modeluser.ModelCredentials, userID string) error
	CheckUser(ctx context.Context, credentials modeluser.ModelCredentials) (string, error)
}

type Storage interface {
	Register
}
