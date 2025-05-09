package models

import (
	"time"
)

type URL struct {
	ID        int       `json:"id" db:"id"`
	Original  string    `json:"original_url" db:"original_url" binding:"required,url"`
	ShortCode string    `json:"short_code" db:"short_code" binding:"required"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type CreateRequest struct {
	URL       string `json:"url" binding:"required,url"`
	ExpiresAt string `json:"expires_at"`
}

type Response struct {
	ShortURL  string    `json:"short_url"`
	ExpiresAt time.Time `json:"expires_at"`
	StatsURL  string    `json:"stats_url"`
}

type Stats struct {
	TotalClicks  int            `json:"total_clicks"`
	LastAccessed time.Time      `json:"last_accessed"`
	DailyClicks  map[string]int `json:"daily_clicks"`
}
