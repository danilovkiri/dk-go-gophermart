package main

import (
	"encoding/json"
	"flag"
	"github.com/ShiraazMoollatjie/goluhn"
	"github.com/caarlos0/env/v6"
	"github.com/danilovkiri/dk-go-gophermart/internal/api/rest/middleware"
	"github.com/go-chi/chi"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

type Response struct {
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

type Order struct {
	Order   string  `json:"order,omitempty"`
	Status  string  `json:"status"`
	Accrual float64 `json:"accrual,omitempty"`
}

type ServerConfig struct {
	ServerAddress string `env:"RUN_ADDRESS"`
}

func NewServerConfig() (*ServerConfig, error) {
	cfg := ServerConfig{}
	err := env.Parse(&cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func isFlagPassed(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

func (c *ServerConfig) ParseFlags() {
	a := flag.String("a", ":7070", "Server address")
	flag.Parse()
	if isFlagPassed("a") || c.ServerAddress == "" {
		c.ServerAddress = *a
	}
}

func HandleMockAccrualServcie() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// mock http status 429 error
		chance429 := 10
		if chance429 > rand.Intn(100) {
			log.Println("responding with error 429")
			w.Header().Set("Retry-After", "60")
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusTooManyRequests)
			response429 := Response{
				Error: "No more than N requests per minute allowed",
			}
			resBody, _ := json.Marshal(response429)
			w.Write(resBody)
			return
		}

		// mock http status 500 error
		chance500 := 20
		if chance500 > rand.Intn(100) {
			log.Println("responding with error 500")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// mock normal behaviour
		orderID := chi.URLParam(r, "orderID")
		orderNumber, err := strconv.Atoi(orderID)
		if err != nil {
			log.Println("responding with error 400")
			w.WriteHeader(http.StatusBadRequest)
			response400 := Response{
				Error: "Invalid order number: not an integer",
			}
			resBody, _ := json.Marshal(response400)
			w.Write(resBody)
			return
		}
		err = goluhn.Validate(orderID)
		if err != nil {
			log.Println("responding with error 422")
			w.WriteHeader(http.StatusUnprocessableEntity)
			response422 := Response{
				Error: "Illegal order number",
			}
			resBody, _ := json.Marshal(response422)
			w.Write(resBody)
			return
		}

		var response200 Order
		switch rand.Intn(4) {
		case 0:
			// PROCESSED
			var accrual float64
			chanceNoAccrual := 5
			if chanceNoAccrual > rand.Intn(10) {
				accrual = 0
			} else {
				accrual = float64(orderNumber%1000) + 0.5
			}
			response200 = Order{
				Order:   orderID,
				Status:  "PROCESSED",
				Accrual: accrual,
			}
		case 1:
			// INVALID
			response200 = Order{
				Order:  orderID,
				Status: "INVALID",
			}
		case 2:
			// REGISTERED
			response200 = Order{
				Order:  orderID,
				Status: "REGISTERED",
			}
		case 3:
			// PROCESSING
			response200 = Order{
				Order:  orderID,
				Status: "PROCESSING",
			}
		}
		log.Println("responding with status 200")
		w.WriteHeader(http.StatusOK)
		resBody, _ := json.Marshal(response200)
		w.Write(resBody)
	}
}

func InitServer(cfg *ServerConfig) (server *http.Server, err error) {
	r := chi.NewRouter()
	r.Use(middleware.CompressHandle)
	r.Use(middleware.DecompressHandle)
	r.Get("/api/orders/{orderID}", HandleMockAccrualServcie())
	srv := &http.Server{
		Addr:         cfg.ServerAddress,
		Handler:      r,
		IdleTimeout:  60 * time.Second,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}
	return srv, nil
}

func main() {
	cfg, err := NewServerConfig()
	if err != nil {
		log.Println(err)
	}
	cfg.ParseFlags()
	server, err := InitServer(cfg)
	if err != nil {
		log.Println(err)
	}
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Println(err)
	}
}
