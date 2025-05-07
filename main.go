package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
	_ "modernc.org/sqlite"
)

type UrlRequest struct {
	URL       string `json:"url" binding:"required,url"`
	ExpiresAt string `json:"expires_at"`
}

type UrlResponse struct {
	ShortURL  string    `json:"short_url"`
	ExpiresAt time.Time `json:"expires_at"`
	StatsURL  string    `json:"stats_url"`
}

type StatsResponse struct {
	TotalClicks  int            `json:"total_clicks"`
	LastAccessed time.Time      `json:"last_accessed"`
	DailyClicks  map[string]int `json:"daily_clicks"`
}

var (
	db          *sql.DB
	rateLimiter = rate.NewLimiter(rate.Every(time.Minute), 10) // 10 requests/min
)

func main() {
	// Configuração inicial
	initDB()
	defer db.Close()

	// Configuração do Gin
	router := gin.Default()

	// Middlewares
	router.Use(corsMiddleware())
	router.Use(rateLimitMiddleware())

	// Rotas
	router.POST("/api/shorten", shortenURL)
	router.GET("/:code", redirectURL)
	router.GET("/api/stats/:code", getStats)

	// Servidor com shutdown graceful
	srv := &http.Server{
		Addr:    ":8000",
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()

	// Capturar sinais de shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		panic(err)
	}
}

func initDB() {
	var err error
	db, err = sql.Open("sqlite", "./urls.db?_foreign_keys=on")
	if err != nil {
		panic(err)
	}

	// Executar migrações
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS urls (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		original_url TEXT NOT NULL,
		short_code TEXT UNIQUE NOT NULL,
		expires_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS stats (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		url_id INTEGER NOT NULL,
		accessed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		ip_address TEXT,
		FOREIGN KEY(url_id) REFERENCES urls(id)
	);`)
	if err != nil {
		panic(err)
	}
}

// Middlewares e helpers
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		c.Next()
	}
}

func rateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rateLimiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests",
			})
		}
		c.Next()
	}
}

func generateCode() string {
	b := make([]byte, 8)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)[:8]
}

// Handlers
func shortenURL(c *gin.Context) {
	var req UrlRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid URL"})
		return
	}

	// Calcular expiração
	expDuration := 24 * time.Hour * 7 // Default 1 semana
	if req.ExpiresAt != "" {
		d, err := time.ParseDuration(req.ExpiresAt)
		if err == nil {
			expDuration = d
		}
	}
	expTime := time.Now().Add(expDuration)

	// Verificar existência
	var existing struct {
		Code    string
		Expires time.Time
	}
	err := db.QueryRowContext(c,
		"SELECT short_code, expires_at FROM urls WHERE original_url = ?",
		req.URL,
	).Scan(&existing.Code, &existing.Expires)

	if err == nil {
		if time.Now().After(existing.Expires) {
			c.JSON(http.StatusGone, gin.H{"error": "URL expired"})
			return
		}
		sendResponse(c, existing.Code, expTime)
		return
	}

	// Gerar novo código
	code := generateCode()
	for {
		var temp string
		err := db.QueryRowContext(c,
			"SELECT short_code FROM urls WHERE short_code = ?",
			code,
		).Scan(&temp)
		if err != nil {
			break
		}
		code = generateCode()
	}

	// Inserir no banco
	_, err = db.ExecContext(c,
		"INSERT INTO urls (original_url, short_code, expires_at) VALUES (?, ?, ?)",
		req.URL, code, expTime,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create URL"})
		return
	}

	sendResponse(c, code, expTime)
}

func redirectURL(c *gin.Context) {
	code := c.Param("code")

	var (
		original string
		expires  time.Time
	)

	err := db.QueryRowContext(c,
		"SELECT original_url, expires_at FROM urls WHERE short_code = ?",
		code,
	).Scan(&original, &expires)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "URL not found"})
		return
	}

	if time.Now().After(expires) {
		c.JSON(http.StatusGone, gin.H{"error": "URL expired"})
		return
	}

	// Registrar acesso
	ip, _, _ := net.SplitHostPort(c.Request.RemoteAddr)
	_, err = db.ExecContext(c,
		"INSERT INTO stats (url_id, ip_address) VALUES ((SELECT id FROM urls WHERE short_code = ?), ?)",
		code, ip,
	)
	if err != nil {
		fmt.Println("Failed to log access:", err)
	}

	c.Redirect(http.StatusFound, original)
}

func getStats(c *gin.Context) {
	code := c.Param("code")

	var stats StatsResponse
	var urlID int

	// Obter ID da URL
	err := db.QueryRowContext(c,
		"SELECT id FROM urls WHERE short_code = ?",
		code,
	).Scan(&urlID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "URL not found"})
		return
	}

	// Total de acessos
	db.QueryRowContext(c,
		"SELECT COUNT(*) FROM stats WHERE url_id = ?",
		urlID,
	).Scan(&stats.TotalClicks)

	// Último acesso
	db.QueryRowContext(c,
		"SELECT accessed_at FROM stats WHERE url_id = ? ORDER BY accessed_at DESC LIMIT 1",
		urlID,
	).Scan(&stats.LastAccessed)

	// Acessos diários
	rows, err := db.QueryContext(c,
		`SELECT DATE(accessed_at) as day, COUNT(*) 
		FROM stats 
		WHERE url_id = ? 
		GROUP BY day`,
		urlID,
	)

	if err == nil {
		defer rows.Close()
		stats.DailyClicks = make(map[string]int)
		for rows.Next() {
			var day string
			var count int
			rows.Scan(&day, &count)
			stats.DailyClicks[day] = count
		}
	}

	c.JSON(http.StatusOK, stats)
}

func sendResponse(c *gin.Context, code string, expires time.Time) {
	c.JSON(http.StatusOK, UrlResponse{
		ShortURL:  fmt.Sprintf("http://%s/%s", c.Request.Host, code),
		ExpiresAt: expires,
		StatsURL:  fmt.Sprintf("http://%s/api/stats/%s", c.Request.Host, code),
	})
}
