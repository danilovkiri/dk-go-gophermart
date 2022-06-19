// Package rest provides functionality for initializing a server
package rest

import (
	"context"
	"github.com/danilovkiri/dk-go-gophermart/internal/api/rest/handlers"
	"github.com/danilovkiri/dk-go-gophermart/internal/api/rest/middleware"
	"github.com/danilovkiri/dk-go-gophermart/internal/config"
	"github.com/danilovkiri/dk-go-gophermart/internal/service/processor/v1/processor"
	"github.com/danilovkiri/dk-go-gophermart/internal/service/secretary/v1/secretary"
	"github.com/danilovkiri/dk-go-gophermart/internal/storage/v1/inpsql"
	"github.com/go-chi/chi"
	"log"
	"net/http"
	"time"
)

// InitServer returns a http.Server object ready to be listening and serving .
func InitServer(ctx context.Context, cfg *config.Config, minorlog *log.Logger) (server *http.Server, err error) {
	// initialize storage
	storage, err := inpsql.InitStorage(ctx, cfg.StorageConfig, minorlog)

	//initialize secretary
	secretaryService, err := secretary.NewSecretaryService(cfg.SecretConfig)
	if err != nil {
		return nil, err
	}

	//initialize cookie handler
	cookieHandler, err := middleware.NewCookieHandler(secretaryService, cfg.SecretConfig)
	if err != nil {
		return nil, err
	}

	// initialize main service
	mainservice, err := processor.InitService(storage, secretaryService)
	if err != nil {
		return nil, err
	}

	urlHandler, err := handlers.InitHandlers(mainservice, cfg.ServerConfig, minorlog)
	if err != nil {
		return nil, err
	}

	r := chi.NewRouter()
	r.Use(middleware.CompressHandle)
	r.Use(middleware.DecompressHandle)
	loginGroup := r.Group(nil)
	loginGroup.Post("/api/user/register", urlHandler.HandleRegister())
	loginGroup.Post("/api/user/login", nil)
	mainGroup := r.Group(nil)
	mainGroup.Use(cookieHandler.CookieHandle)
	mainGroup.Post("/api/user/orders", nil)
	mainGroup.Get("/api/user/orders", nil)
	mainGroup.Get("/api/user/balance", nil)
	mainGroup.Post("/api/user/balance/withdraw", nil)
	mainGroup.Get("/api/user/balance/withdrawals", nil)

	srv := &http.Server{
		Addr:         cfg.ServerConfig.ServerAddress,
		Handler:      r,
		IdleTimeout:  60 * time.Second,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}
	return srv, nil
}
