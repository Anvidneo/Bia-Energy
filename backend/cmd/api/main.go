// Command api is the entrypoint of the Bia Energy backend: loads config,
// connects to Postgres, applies the schema, seeds readings.csv/events.csv
// if the tables are empty, and serves the HTTP API.
package main

import (
	"context"
	"log"
	"net/http"

	"bia-energy/backend/internal/ai"
	"bia-energy/backend/internal/api"
	"bia-energy/backend/internal/config"
	"bia-energy/backend/internal/db"
	"bia-energy/backend/internal/firebase"
	"bia-energy/backend/internal/seed"
)

func main() {
	cfg := config.Load()

	conn, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connecting to database: %v", err)
	}
	defer func() { _ = conn.Close() }()

	if err := db.ApplySchema(conn); err != nil {
		log.Fatalf("applying schema: %v", err)
	}

	if err := seed.Load(conn, "data/readings.csv", "data/events.csv"); err != nil {
		log.Fatalf("seeding database: %v", err)
	}

	var explainer ai.Explainer = ai.RuleExplainer{}
	if cfg.UseLLMExplainer {
		if cfg.LLMAPIKey == "" {
			log.Println("USE_LLM_EXPLAINER=true but LLM_API_KEY is empty; using RuleExplainer")
		} else {
			explainer = ai.NewLLMExplainer(cfg.LLMAPIKey, cfg.LLMModel)
			log.Printf("using ai.LLMExplainer (model=%s)", cfg.LLMModel)
		}
	}

	deps := &api.Deps{DB: conn, Explainer: explainer}

	if cfg.UseFirebaseAlerts {
		if cfg.FirebaseProjectID == "" || cfg.FirebaseCredentialsJSON == "" {
			log.Println("USE_FIREBASE_ALERTS=true but FIREBASE_PROJECT_ID/FIREBASE_CREDENTIALS_JSON are incomplete; alerts disabled")
		} else {
			publisher, err := firebase.NewPublisher(context.Background(), cfg.FirebaseProjectID, cfg.FirebaseCredentialsJSON)
			if err != nil {
				// Firebase must never block startup: log and run without
				// alerts rather than failing the whole service.
				log.Printf("firebase: could not initialize publisher, alerts disabled: %v", err)
			} else {
				defer func() { _ = publisher.Close() }()
				deps.AlertPublisher = publisher
				log.Println("critical alerts will be published to Firebase")
			}
		}
	}

	router := api.NewRouter(deps)

	addr := ":" + cfg.Port
	log.Printf("bia-energy-api listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, router))
}
