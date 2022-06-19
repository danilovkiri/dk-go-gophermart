// Package middleware provides various middleware functionality.
package middleware

import (
	"errors"
	"github.com/danilovkiri/dk-go-gophermart/internal/config"
	"github.com/danilovkiri/dk-go-gophermart/internal/service/secretary/v1"
	"net/http"
)

// CookieHandler sets object structure.
type CookieHandler struct {
	sec secretary.Secretary
	cfg *config.SecretConfig
}

// NewCookieHandler initializes a new cookie handler.
func NewCookieHandler(sec secretary.Secretary, cfg *config.SecretConfig) (*CookieHandler, error) {
	if sec == nil {
		return nil, errors.New("nil secretary object was found")
	}
	return &CookieHandler{
		sec: sec,
		cfg: cfg,
	}, nil
}

// CookieHandle provides cookie handling functionality.
func (c *CookieHandler) CookieHandle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("userID")
		if errors.Is(err, http.ErrNoCookie) {
			//http.Redirect(w, r, "/api/user/login", http.StatusSeeOther)
			http.Error(w, err.Error(), http.StatusUnauthorized)
		} else if err != nil {
			http.Error(w, "Cookie crumbled", http.StatusInternalServerError)
		} else {
			_, err := c.sec.Decode(cookie.Value)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
			}
		}
		next.ServeHTTP(w, r)
	})
}
