// Package config reads runtime configuration from environment variables.
package config

import "os"

// Config holds everything cmd/api/main.go needs to start the service.
type Config struct {
	Port            string
	DatabaseURL     string
	UseLLMExplainer bool
	LLMAPIKey       string
	LLMModel        string

	UseFirebaseAlerts       bool
	FirebaseProjectID       string
	FirebaseCredentialsJSON string
}

// Load reads Config from the environment, applying the same defaults as
// .env.example so `go run ./cmd/api` works out of the box against the
// docker-compose postgres service.
func Load() Config {
	return Config{
		Port:            getEnv("PORT", "8080"),
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://bia:bia@localhost:5432/bia_energy?sslmode=disable"),
		UseLLMExplainer: getEnv("USE_LLM_EXPLAINER", "false") == "true",
		LLMAPIKey:       getEnv("LLM_API_KEY", ""),
		LLMModel:        getEnv("LLM_MODEL", "gemini-2.5-flash"),

		UseFirebaseAlerts:       getEnv("USE_FIREBASE_ALERTS", "false") == "true",
		FirebaseProjectID:       getEnv("FIREBASE_PROJECT_ID", ""),
		FirebaseCredentialsJSON: getEnv("FIREBASE_CREDENTIALS_JSON", ""),
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
