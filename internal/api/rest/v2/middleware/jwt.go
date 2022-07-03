// Package middleware provides various middleware functionality.
package middleware

import (
	"errors"
	"github.com/danilovkiri/dk-go-gophermart/internal/config"
	"github.com/danilovkiri/dk-go-gophermart/internal/service/secretary/v2"
	"net/http"
	"strings"
)

// TokenHandler sets object structure.
type TokenHandler struct {
	sec secretary.Secretary
	cfg *config.SecretConfig
}

// NewTokenHandler initializes a new token handler.
func NewTokenHandler(sec secretary.Secretary, cfg *config.SecretConfig) (*TokenHandler, error) {
	if sec == nil {
		return nil, errors.New("nil secretary object was found")
	}
	return &TokenHandler{
		sec: sec,
		cfg: cfg,
	}, nil
}

// TokenHandle provides token handling functionality.
func (c *TokenHandler) TokenHandle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString := r.Header.Get("Authorization")
		if len(tokenString) == 0 {
			http.Error(w, "Token authorization required", http.StatusUnauthorized)
			return
		}
		tokenString = strings.Replace(tokenString, "Bearer ", "", 1)
		_, err := c.sec.ValidateToken(tokenString)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
