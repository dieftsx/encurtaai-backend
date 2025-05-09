package handlers

import (
	"net/http"
	"urlshortener/internal/database"

	"github.com/gin-gonic/gin"
)

type URLHandler struct {
	db *database.DB
}

func NewURLHandler(db *database.DB) *URLHandler {
	return &URLHandler{db: db}
}

func (h *URLHandler) Shorten(c *gin.Context) {
	var req models.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid request",
		})
		return
	}
}

func (h *URLHandler) Redirect(c *gin.Context) {
	code := c.Param("code")

	var url models.URL
	err := h.db.QueryRowContext(c,
		"SELECT original_url, expires_at FROM urls WHERE short_code = ?",
		code,
	).Scan(&url.Original, &url.ExpiresAt)

	// Resto da implementação
}

// ... outros métodos relacionados a URLs
