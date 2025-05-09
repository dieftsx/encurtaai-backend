package config

import (
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	ServerAddress string `envconfig:"SERVER_ADDRESS" default:":8000"`
	DatabaseURL   string `envconfig:"DATABASE_URL" default:"./urls.db"`
	RateLimit     int    `envconfig:"RATE_LIMIT" default:"10"`
}

func Load() *Config {
	var cfg Config
	err := envconfig.Process("", &cfg)
	if err != nil {
		panic(err)
	}
	return &cfg
}
