// Command api is the entrypoint of the Bia Energy backend: loads config,
// connects to Postgres, applies the schema, seeds readings.csv/events.csv
// if the tables are empty, and serves the HTTP API.
package main

import (
	"log"
	"net/http"

	"bia-energy/backend/internal/ai"
	"bia-energy/backend/internal/api"
	"bia-energy/backend/internal/config"
	"bia-energy/backend/internal/db"
	"bia-energy/backend/internal/seed"
)

func main() {
	cfg := config.Load()

	conn, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connecting to database: %v", err)
	}
	defer conn.Close()

	if err := db.ApplySchema(conn); err != nil {
		log.Fatalf("applying schema: %v", err)
	}

	if err := seed.Load(conn, "data/readings.csv", "data/events.csv"); err != nil {
		log.Fatalf("seeding database: %v", err)
	}

	var explainer ai.Explainer = ai.RuleExplainer{}
	if cfg.UseLLMExplainer {
		// Sunday stretch goal: swap in ai.LLMExplainer here once it
		// exists. Falling back to RuleExplainer keeps the service
		// working even if USE_LLM_EXPLAINER is set before that lands.
		log.Println("USE_LLM_EXPLAINER=true but LLMExplainer isn't built yet; using RuleExplainer")
	}

	deps := &api.Deps{DB: conn, Explainer: explainer}
	router := api.NewRouter(deps)

	addr := ":" + cfg.Port
	log.Printf("bia-energy-api listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, router))
}
