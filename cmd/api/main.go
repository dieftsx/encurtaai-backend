package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
	"urlshortener/internal/config"
	"urlshortener/internal/database"
	"urlshortener/internal/handlers"
)

func main() {
	// Carregar configurações
	cfg := config.Load()

	// Inicializar banco de dados
	db, err := database.Init(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Configurar servidor
	server := handlers.NewServer(db, cfg)

	// Iniciar servidor
	go func() {
		if err := server.Start(cfg.ServerAddress); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	server.Shutdown(5 * time.Second)
}
