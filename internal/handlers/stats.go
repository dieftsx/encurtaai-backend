package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"urlshortener/internal/database"
	"urlshortener/internal/models"

	"github.com/gin-gonic/gin"
)

type StatsHandler struct {
	db *database.DB
}

func NewStatsHandler(db *database.DB) *StatsHandler {
	return &StatsHandler{db: db}
}

func (h *StatsHandler) GetStats(c *gin.Context) {
	code := c.Param("code")
	ctx := c.Request.Context()

	stats, err := h.getStatsData(ctx, code)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, models.Response{
				Code:    http.StatusNotFound,
				Message: "URL not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to get stats",
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *StatsHandler) getStatsData(ctx context.Context, shortCode string) (*models.Stats, error) {
	var stats models.Stats
	var urlID int

	// Get URL ID
	err := h.db.QueryRowContext(ctx,
		"SELECT id FROM urls WHERE short_code = ?",
		shortCode,
	).Scan(&urlID)
	if err != nil {
		return nil, err
	}

	// Total clicks
	err = h.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM stats WHERE url_id = ?",
		urlID,
	).Scan(&stats.TotalClicks)
	if err != nil {
		return nil, err
	}

	// Last accessed
	err = h.db.QueryRowContext(ctx,
		"SELECT accessed_at FROM stats WHERE url_id = ? ORDER BY accessed_at DESC LIMIT 1",
		urlID,
	).Scan(&stats.LastAccessed)

	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Daily clicks
	stats.DailyClicks = make(map[string]int)
	rows, err := h.db.QueryContext(ctx,
		`SELECT DATE(accessed_at) as day, COUNT(*) 
		FROM stats 
		WHERE url_id = ? 
		GROUP BY day`,
		urlID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var day string
		var count int
		if err := rows.Scan(&day, &count); err != nil {
			return nil, err
		}
		stats.DailyClicks[day] = count
	}

	return &stats, nil
}
