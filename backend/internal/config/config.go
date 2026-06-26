// Package config carrega a configuração do serviço a partir de variáveis de
// ambiente. Em dev, o entrypoint carrega o .env da raiz antes de chamar Load.
package config

import (
	"fmt"
	"os"
	"strings"
)

// Config reúne todas as configurações do serviço.
type Config struct {
	AppEnv        string // development | production
	HTTPAddr      string // ex.: ":8080"
	DatabaseURL   string // postgres://...
	BaseURL       string // base para montar o link curto, sem barra final
	APIKey        string // protege os endpoints /api/*
	IPHashSalt    string // salt para hashear o IP do clique
	GeoLiteDBPath string // caminho do GeoLite2-City.mmdb
}

// IsProduction indica se o serviço roda em produção.
func (c *Config) IsProduction() bool {
	return c.AppEnv == "production"
}

// Load lê as variáveis de ambiente e valida as obrigatórias.
func Load() (*Config, error) {
	cfg := &Config{
		AppEnv:        getEnv("APP_ENV", "development"),
		HTTPAddr:      getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		BaseURL:       strings.TrimRight(getEnv("BASE_URL", "http://localhost:8080"), "/"),
		APIKey:        os.Getenv("API_KEY"),
		IPHashSalt:    os.Getenv("IP_HASH_SALT"),
		GeoLiteDBPath: getEnv("GEOLITE_DB_PATH", ""),
	}

	var missing []string
	if cfg.DatabaseURL == "" {
		missing = append(missing, "DATABASE_URL")
	}
	if cfg.APIKey == "" {
		missing = append(missing, "API_KEY")
	}
	if cfg.IPHashSalt == "" {
		missing = append(missing, "IP_HASH_SALT")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("variáveis de ambiente obrigatórias ausentes: %s", strings.Join(missing, ", "))
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
