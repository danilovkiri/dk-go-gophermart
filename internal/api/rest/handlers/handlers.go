package handlers

import (
	"context"
	"encoding/json"
	"errors"
	handlersErrors "github.com/danilovkiri/dk-go-gophermart/internal/api/rest/errors"
	"github.com/danilovkiri/dk-go-gophermart/internal/config"
	"github.com/danilovkiri/dk-go-gophermart/internal/models/modeluser"
	"github.com/danilovkiri/dk-go-gophermart/internal/service/processor/v1"
	storageErrors "github.com/danilovkiri/dk-go-gophermart/internal/storage/v1/errors"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type Handler struct {
	service      processor.Processor
	serverConfig *config.ServerConfig
	localLog     *log.Logger
}

func InitHandlers(mainservice processor.Processor, serverConfig *config.ServerConfig, minorlog *log.Logger) (*Handler, error) {
	if mainservice == nil {
		return nil, &handlersErrors.HandlersFoundNilArgument{Msg: "nil processor was passed to handlers initializer"}
	}
	return &Handler{service: mainservice, serverConfig: serverConfig, localLog: minorlog}, nil
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
			log.Println("HandleRegister:", err)
			h.localLog.Println("HandleRegister:", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var credentials modeluser.ModelCredentials
		err = json.Unmarshal(b, &credentials)
		if err != nil {
			log.Println("HandleRegister:", err)
			h.localLog.Println("HandleRegister:", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		log.Println("New user register request detected for", credentials)
		h.localLog.Println("New user register request detected for", credentials)
		userCookie, err := h.service.AddNewUser(ctx, credentials)
		if err != nil {
			log.Println("HandleRegister:", err)
			h.localLog.Println("HandleRegister:", err)
			var contextTimeoutExceededError *storageErrors.ContextTimeoutExceededError
			var alreadyExistsError *storageErrors.AlreadyExistsError
			if errors.As(err, &contextTimeoutExceededError) {
				http.Error(w, err.Error(), http.StatusGatewayTimeout)
			} else if errors.As(err, &alreadyExistsError) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)
			} else {
				http.Error(w, err.Error(), http.StatusBadRequest)
			}
			return
		}
		http.SetCookie(w, userCookie)
		r.AddCookie(userCookie)
		w.WriteHeader(http.StatusCreated)
	}
}
