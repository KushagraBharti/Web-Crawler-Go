package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	_ "net/http/pprof"

	"webcrawler/internal/api"
	"webcrawler/internal/config"
)

func main() {
	cfg := config.Load()
	runManager := api.NewRunManager(cfg.Defaults, cfg.DataRoot, cfg.SearchBaseURL, cfg.SearchAPIKey)
	server := api.NewServer(runManager, cfg.AllowedOrigin)

	srv := &http.Server{
		Addr:              ":" + itoa(cfg.Port),
		Handler:           server.Router(),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Printf("api listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	log.Printf("shutdown complete")
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
