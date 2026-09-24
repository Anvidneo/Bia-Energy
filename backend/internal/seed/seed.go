package seed

import (
	"database/sql"
	"fmt"
	"os"
)

// Load parses readingsPath/eventsPath and inserts them into db, but only
// if the readings/events tables are still empty — safe to call on every
// startup (docker-compose restarts the backend often).
func Load(db *sql.DB, readingsPath, eventsPath string) error {
	if err := loadReadings(db, readingsPath); err != nil {
		return err
	}
	if err := loadEvents(db, eventsPath); err != nil {
		return err
	}
	return nil
}

func loadReadings(db *sql.DB, path string) error {
	var count int
	if err := db.QueryRow("SELECT count(*) FROM readings").Scan(&count); err != nil {
		return fmt.Errorf("counting readings: %w", err)
	}
	if count > 0 {
		return nil // already seeded
	}

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close()

	readings, err := ParseReadings(f)
	if err != nil {
		return fmt.Errorf("parsing %s: %w", path, err)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("beginning tx: %w", err)
	}
	defer tx.Rollback()

	meters := make(map[string]bool)
	for _, r := range readings {
		if !meters[r.MeterID] {
			if _, err := tx.Exec(`INSERT INTO meters (id) VALUES ($1) ON CONFLICT (id) DO NOTHING`, r.MeterID); err != nil {
				return fmt.Errorf("inserting meter %s: %w", r.MeterID, err)
			}
			meters[r.MeterID] = true
		}
		if _, err := tx.Exec(
			`INSERT INTO readings (meter_id, ts, consumption_kwh, voltage_v, current_a, power_factor)
			 VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT (meter_id, ts) DO NOTHING`,
			r.MeterID, r.Timestamp, r.ConsumptionKWh, r.VoltageV, r.CurrentA, r.PowerFactor,
		); err != nil {
			return fmt.Errorf("inserting reading for %s at %s: %w", r.MeterID, r.Timestamp, err)
		}
	}
	return tx.Commit()
}

func loadEvents(db *sql.DB, path string) error {
	var count int
	if err := db.QueryRow("SELECT count(*) FROM events").Scan(&count); err != nil {
		return fmt.Errorf("counting events: %w", err)
	}
	if count > 0 {
		return nil
	}

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close()

	events, err := ParseEvents(f)
	if err != nil {
		return fmt.Errorf("parsing %s: %w", path, err)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("beginning tx: %w", err)
	}
	defer tx.Rollback()

	for _, e := range events {
		if _, err := tx.Exec(`INSERT INTO meters (id) VALUES ($1) ON CONFLICT (id) DO NOTHING`, e.MeterID); err != nil {
			return fmt.Errorf("inserting meter %s: %w", e.MeterID, err)
		}
		if _, err := tx.Exec(
			`INSERT INTO events (meter_id, event_ts, event_type, description)
			 VALUES ($1,$2,$3,$4) ON CONFLICT (meter_id, event_ts, event_type) DO NOTHING`,
			e.MeterID, e.EventTimestamp, e.EventType, e.Description,
		); err != nil {
			return fmt.Errorf("inserting event for %s at %s: %w", e.MeterID, e.EventTimestamp, err)
		}
	}
	return tx.Commit()
}
