package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	// No relevant env vars set: Load must fall back to the same defaults
	// documented in .env.example so `go run ./cmd/api` works out of the box.
	t.Setenv("PORT", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("USE_LLM_EXPLAINER", "")

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
}

func TestLoadReadsEnvOverrides(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("DATABASE_URL", "postgres://custom/db")
	t.Setenv("USE_LLM_EXPLAINER", "true")

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
