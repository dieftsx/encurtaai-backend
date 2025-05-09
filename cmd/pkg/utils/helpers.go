// pkg/utils/helpers.go
package utils

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/url"
	"time"
)

// Gera código curto seguro para URLs
func GenerateShortCode(length int) (string, error) {
	if length < 4 {
		return "", fmt.Errorf("length deve ser pelo menos 4")
	}

	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("erro ao gerar código: %w", err)
	}

	// Usar URLEncoding para garantir caracteres seguros
	return base64.URLEncoding.EncodeToString(b)[:length], nil
}

// ParseDuration seguro com fallback
func ParseDurationWithDefault(durationStr string, defaultDuration time.Duration) time.Duration {
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		return defaultDuration
	}
	return duration
}

// Validação de URL
func ValidateURL(rawURL string) bool {
	u, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return false
	}
	return u.Scheme != "" && u.Host != ""
}

// Handler de erros genérico
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func NewError(code int, message string) *ErrorResponse {
	return &ErrorResponse{
		Code:    code,
		Message: message,
	}
}

// Logger simplificado
func LogError(context string, err error) {
	fmt.Printf("[%s] ERROR: %v\n", context, err)
}

// Paginação
type Pagination struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
}

func NewPagination(page, limit int) Pagination {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return Pagination{
		Page:  page,
		Limit: limit,
	}
}
