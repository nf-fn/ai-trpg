package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nf-fn/ai-trpg/internal/gm"
	"github.com/nf-fn/ai-trpg/internal/ollama"
	"github.com/nf-fn/ai-trpg/internal/server"
	"github.com/nf-fn/ai-trpg/internal/voicevox"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := server.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	ollamaClient := ollama.NewClient(cfg.Ollama.URL, cfg.Ollama.Model, cfg.Ollama.Timeout)
	voicevoxClient := voicevox.NewClient(cfg.Voicevox.URL, cfg.Voicevox.Speaker)
	gmEngine := gm.NewEngine(ollamaClient)

	srv := server.New(cfg, ollamaClient, voicevoxClient, gmEngine)

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: srv.Handler(),
	}

	go func() {
		log.Printf("server starting on http://localhost:%d", cfg.Server.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("server shutdown error: %v", err)
	}
	log.Println("server stopped")
}
