package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	handlersErrors "github.com/danilovkiri/dk-go-gophermart/internal/api/rest/errors"
	"github.com/danilovkiri/dk-go-gophermart/internal/config"
	"github.com/danilovkiri/dk-go-gophermart/internal/models/modeluser"
	"github.com/danilovkiri/dk-go-gophermart/internal/service/processor/v1"
	storageErrors "github.com/danilovkiri/dk-go-gophermart/internal/storage/v1/errors"
	"github.com/rs/zerolog"
	"io/ioutil"
	"net/http"
	"time"
)

type Handler struct {
	service      processor.Processor
	serverConfig *config.ServerConfig
	log          *zerolog.Logger
}

func InitHandlers(mainService processor.Processor, serverConfig *config.ServerConfig, log *zerolog.Logger) (*Handler, error) {
	if mainService == nil {
		return nil, &handlersErrors.HandlersFoundNilArgument{Msg: "nil processor was passed to handlers initializer"}
	}
	return &Handler{service: mainService, serverConfig: serverConfig, log: log}, nil
}

func (h *Handler) HandleRegister() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
		defer cancel()
		if r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, "Invalid Content-Type", http.StatusBadRequest)
		}
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			h.log.Error().Err(err).Msg("handle register failed")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var credentials modeluser.ModelCredentials
		err = json.Unmarshal(b, &credentials)
		if err != nil {
			h.log.Error().Err(err).Msg("handle register failed")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		h.log.Info().Msg(fmt.Sprintf("new user register request detected for %s", credentials))
		userCookie, err := h.service.AddNewUser(ctx, credentials)
		if err != nil {
			h.log.Error().Err(err).Msg("handle register failed")
			var contextTimeoutExceededError *storageErrors.ContextTimeoutExceededError
			var alreadyExistsError *storageErrors.AlreadyExistsError
			if errors.As(err, &contextTimeoutExceededError) {
				http.Error(w, err.Error(), http.StatusGatewayTimeout)
			} else if errors.As(err, &alreadyExistsError) {
				w.WriteHeader(http.StatusConflict)
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
		http.SetCookie(w, userCookie)
		r.AddCookie(userCookie)
		w.WriteHeader(http.StatusOK)
	}
}

func (h *Handler) HandleLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
		defer cancel()
		if r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, "Invalid Content-Type", http.StatusBadRequest)
		}
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			h.log.Error().Err(err).Msg("handle login failed")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var credentials modeluser.ModelCredentials
		err = json.Unmarshal(b, &credentials)
		if err != nil {
			h.log.Error().Err(err).Msg("handle login failed")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		h.log.Info().Msg(fmt.Sprintf("new login request detected for %s", credentials))
		userCookie, err := h.service.LoginUser(ctx, credentials)
		if err != nil {
			h.log.Error().Err(err).Msg("handle login failed")
			var contextTimeoutExceededError *storageErrors.ContextTimeoutExceededError
			var notFoundError *storageErrors.NotFoundError
			if errors.As(err, &contextTimeoutExceededError) {
				http.Error(w, err.Error(), http.StatusGatewayTimeout)
			} else if errors.As(err, &notFoundError) {
				w.WriteHeader(http.StatusUnauthorized)
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
		http.SetCookie(w, userCookie)
		r.AddCookie(userCookie)
		w.WriteHeader(http.StatusOK)
	}
}
