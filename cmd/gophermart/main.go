package main

import (
	"context"
	"github.com/danilovkiri/dk-go-gophermart/internal/api/rest"
	"github.com/danilovkiri/dk-go-gophermart/internal/config"
	"github.com/danilovkiri/dk-go-gophermart/internal/logger"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {
	wg := &sync.WaitGroup{}

	log := logger.InitLog()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// get configuration
	cfg, err := config.NewConfiguration()
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}
	cfg.ParseFlags()

	// initialize server
	server, err := rest.InitServer(ctx, cfg, log, wg)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	// set a listener for graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-done
		log.Info().Msg("server shutdown attempted")
		ctxTO, cancelTO := context.WithTimeout(ctx, 5*time.Second)
		defer cancelTO()
		if err := server.Shutdown(ctxTO); err != nil {
			log.Fatal().Err(err).Msg("server shutdown failed")
		}
		cancel()
	}()

	// start up the server
	log.Info().Msg("server start attempted")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("")
	}
	wg.Wait()
	log.Info().Msg("server shutdown succeeded")
}
