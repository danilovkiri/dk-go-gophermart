package main

import (
	"context"
	"github.com/danilovkiri/dk-go-gophermart/internal/api/rest"
	"github.com/danilovkiri/dk-go-gophermart/internal/config"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// make a top-level file logger for logging critical errors
	flog, err := os.OpenFile(`server.log`, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer flog.Close()
	majorlog := log.New(flog, `major `, log.LstdFlags|log.Lshortfile)
	minorlog := log.New(flog, `minor `, log.LstdFlags|log.Lshortfile)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// get configuration
	cfg, err := config.NewConfiguration()
	if err != nil {
		log.Println(err)
		majorlog.Fatal(err)
	}
	cfg.ParseFlags()

	// initialize server
	server, err := rest.InitServer(ctx, cfg, minorlog)
	if err != nil {
		log.Println(err)
		majorlog.Fatal(err)
	}

	// set a listener for graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-done
		log.Println("Server shutdown attempted")
		majorlog.Print("Server shutdown attempted")
		ctxTO, cancelTO := context.WithTimeout(ctx, 5*time.Second)
		defer cancelTO()
		if err := server.Shutdown(ctxTO); err != nil {
			log.Println("Server shutdown failed:", err)
			majorlog.Fatal("Server shutdown failed:", err)
		}
		cancel()
	}()

	// start up the server
	log.Println("Server start attempted")
	majorlog.Print("Server start attempted")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Println(err)
		majorlog.Fatal(err)
	}
	//wg.Wait()
	log.Println("Server shutdown succeeded")
	majorlog.Print("Server shutdown succeeded")
}
