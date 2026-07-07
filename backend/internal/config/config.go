package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	DatabaseURL   string
	HTTPAddr      string
	JWTSecret     string
	TokenTTL      time.Duration
	MigrationsDir string
	FileStoreDir  string
	CORSOrigin    string // dev frontend origin, empty disables CORS headers

	// LLM-provider (OpenAI-kompatibelt API — DeepSeek, gateways m.fl.).
	// Tom LLMBaseURL = ingen provider; assistenten degraderer pænt.
	LLMBaseURL string
	LLMAPIKey  string
	LLMModel   string
	AIDocsDir  string // redigerbare instruktionsfiler (MD) til AI-funktioner

	// Luftfoto-WMS (standard: Dataforsyningens frie danske ortofoto).
	OrthoWMSURL string
	OrthoLayer  string
	OrthoToken  string

	// Frie, nøglefri opslagstjenester: adresser (DAWA) og lokalplaner
	// (Plandata WFS).
	DAWABaseURL    string
	PlandataWFSURL string
}

// FromEnv reads configuration from environment variables. JWT_SECRET and
// DATABASE_URL are required; everything else has sensible defaults.
func FromEnv() (Config, error) {
	cfg := Config{
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		HTTPAddr:      getenv("HTTP_ADDR", ":8080"),
		JWTSecret:     os.Getenv("JWT_SECRET"),
		TokenTTL:      30 * 24 * time.Hour,
		MigrationsDir: getenv("MIGRATIONS_DIR", "db/migrations"),
		FileStoreDir:  getenv("FILE_STORE_DIR", "data/files"),
		CORSOrigin:    os.Getenv("CORS_ORIGIN"),
		LLMBaseURL:    os.Getenv("LLM_BASE_URL"), // fx https://api.deepseek.com/v1
		LLMAPIKey:     os.Getenv("LLM_API_KEY"),
		LLMModel:      getenv("LLM_MODEL", "deepseek-chat"),
		AIDocsDir:     getenv("AI_DOCS_DIR", "ai-docs"),
		OrthoWMSURL:   getenv("ORTHO_WMS_URL", "https://api.dataforsyningen.dk/orto_foraar_DAF"),
		OrthoLayer:    getenv("ORTHO_WMS_LAYER", "orto_foraar"),
		OrthoToken:    os.Getenv("ORTHO_TOKEN"),
		DAWABaseURL:   getenv("DAWA_BASE_URL", "https://api.dataforsyningen.dk"),
		PlandataWFSURL: getenv("PLANDATA_WFS_URL", "https://geoserver.plandata.dk/geoserver/wfs"),
	}
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return Config{}, fmt.Errorf("JWT_SECRET is required")
	}
	if ttl := os.Getenv("TOKEN_TTL"); ttl != "" {
		d, err := time.ParseDuration(ttl)
		if err != nil {
			return Config{}, fmt.Errorf("invalid TOKEN_TTL: %w", err)
		}
		cfg.TokenTTL = d
	}
	return cfg, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
