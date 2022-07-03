// Package handlers provides API endpoint handling functionality.

package handlers

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	handlersErrors "github.com/danilovkiri/dk-go-gophermart/internal/api/rest/v1/errors"
	"github.com/danilovkiri/dk-go-gophermart/internal/config"
	"github.com/danilovkiri/dk-go-gophermart/internal/models/modeldto"
	"github.com/danilovkiri/dk-go-gophermart/internal/service/processor/v1"
	serviceErrors "github.com/danilovkiri/dk-go-gophermart/internal/service/processor/v1/errors"
	storageErrors "github.com/danilovkiri/dk-go-gophermart/internal/storage/v1/errors"
	"github.com/rs/zerolog"
	"io/ioutil"
	"net/http"
	"time"
)

// Handler defines attributes of a struct available to its methods.
type Handler struct {
	service      processor.Processor
	serverConfig *config.ServerConfig
	log          *zerolog.Logger
}

// InitHandlers initializes a handler object.
func InitHandlers(mainService processor.Processor, serverConfig *config.ServerConfig, log *zerolog.Logger) (*Handler, error) {
	if mainService == nil {
		return nil, &handlersErrors.HandlersFoundNilArgument{Msg: "nil processor was passed to handlers initializer"}
	}
	return &Handler{service: mainService, serverConfig: serverConfig, log: log}, nil
}

// HandleRegister processes user register requests.
func (h *Handler) HandleRegister() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
		defer cancel()
		if r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, "Invalid Content-Type", http.StatusBadRequest)
		}
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			h.log.Error().Err(err).Msg("HandleRegister failed")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var credentials modeldto.User
		err = json.Unmarshal(b, &credentials)
		if err != nil {
			h.log.Error().Err(err).Msg("HandleRegister failed")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		h.log.Info().Msg(fmt.Sprintf("new user register request detected for %s", credentials))
		if len(credentials.Login) == 0 || len(credentials.Password) == 0 {
			h.log.Error().Msg("HandleRegister failed")
			http.Error(w, "Empty values are not allowed", http.StatusBadRequest)
			return
		}
		userCookie, err := h.service.AddNewUser(ctx, credentials)
		if err != nil {
			h.log.Error().Err(err).Msg("HandleRegister failed")
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

// HandleLogin processes user login requests.
func (h *Handler) HandleLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
		defer cancel()
		if r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, "Invalid Content-Type", http.StatusBadRequest)
		}
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			h.log.Error().Err(err).Msg("HandleLogin failed")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var credentials modeldto.User
		err = json.Unmarshal(b, &credentials)
		if err != nil {
			h.log.Error().Err(err).Msg("HandleLogin failed")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		h.log.Info().Msg(fmt.Sprintf("new login request detected for %s", credentials))
		if credentials.Login == "" || credentials.Password == "" {
			h.log.Error().Msg("HandleRegister failed")
			http.Error(w, "Empty values are not allowed", http.StatusBadRequest)
			return
		}
		userCookie, err := h.service.LoginUser(ctx, credentials)
		if err != nil {
			h.log.Error().Err(err).Msg("HandleLogin failed")
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

// HandleGetBalance processes balance query requests.
func (h *Handler) HandleGetBalance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
		defer cancel()
		cipheredUserID, err := getUserID(r)
		if err != nil {
			h.log.Error().Err(err).Msg("HandleBalance failed")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		balance, err := h.service.GetBalance(ctx, cipheredUserID)
		if err != nil {
			h.log.Error().Err(err).Msg("HandleBalance failed")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		resBody, err := json.Marshal(balance)
		if err != nil {
			h.log.Error().Err(err).Msg("HandleBalance failed")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(resBody)
		if err != nil {
			h.log.Error().Err(err).Msg("HandleBalance failed")
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// HandleGetWithdrawals processes withdrawals query requests.
func (h *Handler) HandleGetWithdrawals() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
		defer cancel()
		cipheredUserID, err := getUserID(r)
		if err != nil {
			h.log.Error().Err(err).Msg("HandleWithdrawals failed")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		withdrawals, err := h.service.GetWithdrawals(ctx, cipheredUserID)
		if err != nil {
			h.log.Error().Err(err).Msg("HandleBalance failed")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if len(withdrawals) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		resBody, err := json.Marshal(withdrawals)
		if err != nil {
			h.log.Error().Err(err).Msg("HandleWithdrawals failed")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(resBody)
		if err != nil {
			h.log.Error().Err(err).Msg("HandleWithdrawals failed")
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// HandleGetOrders processes orders query requests.
func (h *Handler) HandleGetOrders() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
		defer cancel()
		cipheredUserID, err := getUserID(r)
		if err != nil {
			h.log.Error().Err(err).Msg("HandleGetOrders failed")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		orders, err := h.service.GetOrders(ctx, cipheredUserID)
		if err != nil {
			h.log.Error().Err(err).Msg("HandleGetOrders failed")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if len(orders) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		resBody, err := json.Marshal(orders)
		if err != nil {
			h.log.Error().Err(err).Msg("HandleGetOrders failed")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(resBody)
		if err != nil {
			h.log.Error().Err(err).Msg("HandleGetOrders failed")
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// HandleNewWithdrawal processes new withdrawal requests.
func (h *Handler) HandleNewWithdrawal() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
		defer cancel()
		cipheredUserID, err := getUserID(r)
		if err != nil {
			h.log.Error().Err(err).Msg("HandleGetOrders failed")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, "Invalid Content-Type", http.StatusBadRequest)
		}
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			h.log.Error().Err(err).Msg("HandleNewWithdrawal failed")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var newOrderWithdrawal modeldto.NewOrderWithdrawal
		err = json.Unmarshal(b, &newOrderWithdrawal)
		if err != nil {
			h.log.Error().Err(err).Msg("HandleNewWithdrawal failed")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		h.log.Info().Msg(fmt.Sprintf("new withdrawal request detected for %v", newOrderWithdrawal))
		err = h.service.AddNewWithdrawal(ctx, cipheredUserID, newOrderWithdrawal)
		if err != nil {
			h.log.Error().Err(err).Msg("HandleNewWithdrawal failed")
			var contextTimeoutExceededError *storageErrors.ContextTimeoutExceededError
			var alreadyExistsError *storageErrors.AlreadyExistsError
			var serviceIllegalOrderNumber *serviceErrors.ServiceIllegalOrderNumber
			var serviceNotEnoughFunds *serviceErrors.ServiceNotEnoughFunds
			if errors.As(err, &contextTimeoutExceededError) {
				http.Error(w, err.Error(), http.StatusGatewayTimeout)
			} else if errors.As(err, &serviceIllegalOrderNumber) || errors.As(err, &alreadyExistsError) {
				w.WriteHeader(http.StatusUnprocessableEntity)
			} else if errors.As(err, &serviceNotEnoughFunds) {
				http.Error(w, err.Error(), http.StatusPaymentRequired)
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// HandleNewOrder processes new order requests.
func (h *Handler) HandleNewOrder() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
		defer cancel()
		cipheredUserID, err := getUserID(r)
		if err != nil {
			h.log.Error().Err(err).Msg("HandleNewOrder failed")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if r.Header.Get("Content-Type") != "text/plain" {
			http.Error(w, "Invalid Content-Type", http.StatusBadRequest)
		}
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			h.log.Error().Err(err).Msg("HandleNewOrder failed")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		orderNumber := string(b)
		h.log.Info().Msg(fmt.Sprintf("new order request detected for order %s", orderNumber))
		err = h.service.AddNewOrder(ctx, cipheredUserID, orderNumber)
		if err != nil {
			h.log.Error().Err(err).Msg("HandleNewWithdrawal failed")
			var contextTimeoutExceededError *storageErrors.ContextTimeoutExceededError
			var alreadyExistsError *storageErrors.AlreadyExistsError
			var alreadyExistsAndViolatesError *storageErrors.AlreadyExistsAndViolatesError
			var serviceIllegalOrderNumber *serviceErrors.ServiceIllegalOrderNumber
			if errors.As(err, &contextTimeoutExceededError) {
				http.Error(w, err.Error(), http.StatusGatewayTimeout)
			} else if errors.As(err, &serviceIllegalOrderNumber) {
				w.WriteHeader(http.StatusUnprocessableEntity)
			} else if errors.As(err, &alreadyExistsError) {
				w.WriteHeader(http.StatusOK)
			} else if errors.As(err, &alreadyExistsAndViolatesError) {
				w.WriteHeader(http.StatusConflict)
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}
}

// getUserID retrieves user identifier from the request metadata.
func getUserID(r *http.Request) (string, error) {
	userCookie, err := r.Cookie("userID")
	if err != nil {
		return "", err
	}
	token := userCookie.Value
	data, err := hex.DecodeString(token)
	if err != nil {
		return "", err
	}
	userID := data
	return hex.EncodeToString(userID), nil
}
