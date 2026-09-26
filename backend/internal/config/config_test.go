package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	// No relevant env vars set: Load must fall back to the same defaults
	// documented in .env.example so `go run ./cmd/api` works out of the box.
	t.Setenv("PORT", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("USE_LLM_EXPLAINER", "")
	t.Setenv("LLM_API_KEY", "")
	t.Setenv("LLM_MODEL", "")
	t.Setenv("USE_FIREBASE_ALERTS", "")
	t.Setenv("FIREBASE_PROJECT_ID", "")
	t.Setenv("FIREBASE_CREDENTIALS_JSON", "")

	cfg := Load()

	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want 8080", cfg.Port)
	}
	if cfg.DatabaseURL != "postgres://bia:bia@localhost:5432/bia_energy?sslmode=disable" {
		t.Errorf("DatabaseURL = %q, want the documented default", cfg.DatabaseURL)
	}
	if cfg.UseLLMExplainer {
		t.Error("UseLLMExplainer = true, want false by default")
	}
	if cfg.LLMAPIKey != "" {
		t.Errorf("LLMAPIKey = %q, want empty by default", cfg.LLMAPIKey)
	}
	if cfg.LLMModel != "gemini-2.5-flash" {
		t.Errorf("LLMModel = %q, want the documented default", cfg.LLMModel)
	}
	if cfg.UseFirebaseAlerts {
		t.Error("UseFirebaseAlerts = true, want false by default")
	}
	if cfg.FirebaseProjectID != "" {
		t.Errorf("FirebaseProjectID = %q, want empty by default", cfg.FirebaseProjectID)
	}
	if cfg.FirebaseCredentialsJSON != "" {
		t.Errorf("FirebaseCredentialsJSON = %q, want empty by default", cfg.FirebaseCredentialsJSON)
	}
}

func TestLoadReadsEnvOverrides(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("DATABASE_URL", "postgres://custom/db")
	t.Setenv("USE_LLM_EXPLAINER", "true")
	t.Setenv("LLM_API_KEY", "test-key")
	t.Setenv("LLM_MODEL", "gemini-custom")
	t.Setenv("USE_FIREBASE_ALERTS", "true")
	t.Setenv("FIREBASE_PROJECT_ID", "bia-energy-project")
	t.Setenv("FIREBASE_CREDENTIALS_JSON", `{"type":"service_account"}`)

	cfg := Load()

	if cfg.Port != "9090" {
		t.Errorf("Port = %q, want 9090", cfg.Port)
	}
	if cfg.DatabaseURL != "postgres://custom/db" {
		t.Errorf("DatabaseURL = %q, want postgres://custom/db", cfg.DatabaseURL)
	}
	if !cfg.UseLLMExplainer {
		t.Error("UseLLMExplainer = false, want true when USE_LLM_EXPLAINER=true")
	}
	if cfg.LLMAPIKey != "test-key" {
		t.Errorf("LLMAPIKey = %q, want test-key", cfg.LLMAPIKey)
	}
	if cfg.LLMModel != "gemini-custom" {
		t.Errorf("LLMModel = %q, want gemini-custom", cfg.LLMModel)
	}
	if !cfg.UseFirebaseAlerts {
		t.Error("UseFirebaseAlerts = false, want true when USE_FIREBASE_ALERTS=true")
	}
	if cfg.FirebaseProjectID != "bia-energy-project" {
		t.Errorf("FirebaseProjectID = %q, want bia-energy-project", cfg.FirebaseProjectID)
	}
	if cfg.FirebaseCredentialsJSON != `{"type":"service_account"}` {
		t.Errorf("FirebaseCredentialsJSON = %q, want the raw JSON", cfg.FirebaseCredentialsJSON)
	}
}

func TestLoadUseLLMExplainerOnlyTrueForExactMatch(t *testing.T) {
	// getEnv falls back on an empty string but not on other unset-ish
	// values — any value other than the literal "true" must mean false.
	for _, v := range []string{"TRUE", "1", "yes", "false", "garbage"} {
		t.Setenv("USE_LLM_EXPLAINER", v)
		if cfg := Load(); cfg.UseLLMExplainer {
			t.Errorf("USE_LLM_EXPLAINER=%q => UseLLMExplainer = true, want false", v)
		}
	}
}

func TestLoadUseFirebaseAlertsOnlyTrueForExactMatch(t *testing.T) {
	for _, v := range []string{"TRUE", "1", "yes", "false", "garbage"} {
		t.Setenv("USE_FIREBASE_ALERTS", v)
		if cfg := Load(); cfg.UseFirebaseAlerts {
			t.Errorf("USE_FIREBASE_ALERTS=%q => UseFirebaseAlerts = true, want false", v)
		}
	}
}
