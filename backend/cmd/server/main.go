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
	var store storage.Store
	storageMode := "memory"
	if cfg.DisableDB || cfg.DatabaseURL == "" {
		store = storage.NewMemory()
		log.Printf("running in memory-only mode (no database)")
	} else {
		db, err := sql.Open("pgx", cfg.DatabaseURL)
		if err != nil {
			log.Printf("db open failed (%v); falling back to memory store", err)
			store = storage.NewMemory()
		} else {
			defer db.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			sqlStore := storage.NewSQL(db)
			if err := sqlStore.Migrate(ctx); err != nil {
				log.Printf("migrate failed (%v); falling back to memory store", err)
				store = storage.NewMemory()
			} else {
				store = sqlStore
				storageMode = "postgres"
			}
		}
	}

	runManager := api.NewRunManager(store, cfg.Defaults)
	server := api.NewServer(runManager, cfg.AllowedOrigin, storageMode)

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
