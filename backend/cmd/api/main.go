// Command api is the entrypoint of the Bia Energy backend.
//
// This is today's (Thursday) skeleton only: it exposes a single health
// check so docker-compose and CI have something real to build and run
// against. Friday's work wires this up to internal/config, internal/db
// (connect + seed) and mounts the internal/api router with the real
// endpoints.
package main

import (
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","service":"bia-energy-api"}`))
	})

	addr := ":8080"
	log.Printf("bia-energy-api skeleton listening on %s (health check only)", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
