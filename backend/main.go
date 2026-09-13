package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"demo-mf-app/internal/api"
	"demo-mf-app/internal/config"
	"demo-mf-app/internal/mfatlas"
	"demo-mf-app/internal/reconcile"
	"demo-mf-app/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := store.Connect(ctx, cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		log.Fatalf("mongo connect: %v", err)
	}
	defer db.Disconnect(context.Background())

	mfClient := mfatlas.NewClient(cfg.MFAtlasBaseURL, cfg.MFAtlasClientID, cfg.MFAtlasClientSecret)

	app := &api.App{Cfg: cfg, MF: mfClient, Store: db}

	frontendDir := os.Getenv("FRONTEND_DIR")
	if frontendDir == "" {
		frontendDir = filepath.Join("..", "frontend")
		if _, err := os.Stat(frontendDir); err != nil {
			frontendDir = "frontend"
		}
	}

	handler := api.NewRouter(app, frontendDir)

	poller := &reconcile.Poller{MF: mfClient, Store: db}
	go poller.Run(ctx, cfg.ReconcileInterval)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Printf("serving on :%s (frontend dir: %s)", cfg.Port, frontendDir)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server: %v", err)
	}
}
