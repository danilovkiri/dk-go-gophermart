// Package rest provides functionality for initializing a server
package rest

import (
	"context"
	"github.com/danilovkiri/dk-go-gophermart/internal/api/rest/client"
	"github.com/danilovkiri/dk-go-gophermart/internal/api/rest/handlers"
	"github.com/danilovkiri/dk-go-gophermart/internal/api/rest/middleware"
	"github.com/danilovkiri/dk-go-gophermart/internal/config"
	"github.com/danilovkiri/dk-go-gophermart/internal/service/broker/v1/broker"
	"github.com/danilovkiri/dk-go-gophermart/internal/service/processor/v1/processor"
	"github.com/danilovkiri/dk-go-gophermart/internal/service/secretary/v1/secretary"
	"github.com/danilovkiri/dk-go-gophermart/internal/storage/v1/inpsql"
	"github.com/go-chi/chi"
	"github.com/rs/zerolog"
	"net/http"
	"sync"
	"time"
)

// InitServer returns a http.Server object ready to be listening and serving .
func InitServer(ctx context.Context, cfg *config.Config, log *zerolog.Logger, wg *sync.WaitGroup) (server *http.Server, err error) {
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

	// initialize storage
	storage, err := inpsql.InitStorage(ctx, cfg.StorageConfig, log, wg)

	// initialize main service
	mainService, err := processor.InitService(storage, secretaryService)
	if err != nil {
		return nil, err
	}

	// initialize accrual client
	brokerClient := client.InitClient(cfg.ServerConfig, log)

	// initialize broker
	brokerService := broker.InitBroker(ctx, storage.QueueIn, storage.QueueOut, log, wg, brokerClient, cfg.QueueConfig.WorkerNumber)
	brokerService.ListenAndProcess()

	urlHandler, err := handlers.InitHandlers(mainService, cfg.ServerConfig, log)
	if err != nil {
		return nil, err
	}

	r := chi.NewRouter()
	r.Use(middleware.CompressHandle)
	r.Use(middleware.DecompressHandle)
	loginGroup := r.Group(nil)
	mainGroup := r.Group(nil)
	mainGroup.Use(cookieHandler.CookieHandle)
	loginGroup.Post("/api/user/register", urlHandler.HandleRegister())
	loginGroup.Post("/api/user/login", urlHandler.HandleLogin())
	mainGroup.Post("/api/user/orders", urlHandler.HandleNewOrder())
	mainGroup.Get("/api/user/orders", urlHandler.HandleGetOrders())
	mainGroup.Get("/api/user/balance", urlHandler.HandleGetBalance())
	mainGroup.Post("/api/user/balance/withdraw", urlHandler.HandleNewWithdrawal())
	mainGroup.Get("/api/user/balance/withdrawals", urlHandler.HandleGetWithdrawals())

	srv := &http.Server{
		Addr:         cfg.ServerConfig.ServerAddress,
		Handler:      r,
		IdleTimeout:  60 * time.Second,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}
	return srv, nil
}
