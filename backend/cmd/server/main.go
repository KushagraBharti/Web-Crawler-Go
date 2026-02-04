package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "net/http/pprof"

	"webcrawler/internal/api"
	"webcrawler/internal/config"
	"webcrawler/internal/storage"
)

func main() {
	cfg := config.Load()

	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	store := storage.New(db)
	if err := store.Migrate(ctx); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	runManager := api.NewRunManager(store, cfg.Defaults)
	server := api.NewServer(runManager, cfg.AllowedOrigin)

	srv := &http.Server{
		Addr:    ":" + itoa(cfg.Port),
		Handler: server.Router(),
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

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = srv.Shutdown(shutdownCtx)
	log.Printf("shutdown complete")
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
