package handlers

import (
	"context"
	"time"
	"urlshortener/internal/config"
	"urlshortener/internal/database"
	"urlshortener/internal/middleware"
     "urlshortener/internal/models"
	"github.com/gin-gonic/gin"
)

type Server struct {
	router *gin.Engine
	db     *database.DB
}

func NewServer(db *database.DB, cfg *config.Config) *Server {
	router := gin.Default()

	// Middlewares
	router.Use(middleware.CORS())
	router.Use(middleware.RateLimit(cfg.RateLimit))

	// Handlers
	urlHandler := NewURLHandler(db)
	statsHandler := NewStatsHandler(db)

	// Rotas
	api := router.Group("/api")
	{
		api.POST("/shorten", urlHandler.Shorten)
		api.GET("/stats/:code", statsHandler.GetStats)
	}

	router.GET("/:code", urlHandler.Redirect)

	return &Server{
		router: router,
		db:     db,
	}
}

func (s *Server) Start(address string) error {
	return s.router.Run(address)
}

func (s *Server) Shutdown(timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Fechar conexões do banco
	s.db.Close()

	// Desligar servidor HTTP
	s.router.Server.Shutdown(ctx)
}
