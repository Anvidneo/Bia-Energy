// Package seed parses readings.csv and events.csv and, once a database is
// wired up (see Load), inserts them if the tables are empty (idempotent).
//
// The parsing functions here are pure (io.Reader in, structs out) on
// purpose: internal/detection's tests load the same real CSVs through
// these functions without needing a database.
package seed

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"bia-energy/backend/internal/models"
)

const timeLayout = "2006-01-02 15:04:05"
const timeLayoutNoSeconds = "2006-01-02 15:04"

func parseTimestamp(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if t, err := time.Parse(timeLayout, s); err == nil {
		return t, nil
	}
	if t, err := time.Parse(timeLayoutNoSeconds, s); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("unrecognized timestamp %q", s)
}

// ParseReadings reads readings.csv:
// meter_id,timestamp,consumption_kwh,voltage_v,current_a,power_factor,status
func ParseReadings(r io.Reader) ([]models.Reading, error) {
	cr := csv.NewReader(r)
	header, err := cr.Read()
	if err != nil {
		return nil, fmt.Errorf("reading header: %w", err)
	}
	idx, err := columnIndex(header, "meter_id", "timestamp", "consumption_kwh", "voltage_v", "current_a", "power_factor")
	if err != nil {
		return nil, err
	}

	var out []models.Reading
	for {
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading row: %w", err)
		}
		ts, err := parseTimestamp(rec[idx["timestamp"]])
		if err != nil {
			return nil, err
		}
		consumption, err := strconv.ParseFloat(strings.TrimSpace(rec[idx["consumption_kwh"]]), 64)
		if err != nil {
			return nil, fmt.Errorf("parsing consumption_kwh: %w", err)
		}
		voltage, err := strconv.ParseFloat(strings.TrimSpace(rec[idx["voltage_v"]]), 64)
		if err != nil {
			return nil, fmt.Errorf("parsing voltage_v: %w", err)
		}
		current, err := strconv.ParseFloat(strings.TrimSpace(rec[idx["current_a"]]), 64)
		if err != nil {
			return nil, fmt.Errorf("parsing current_a: %w", err)
		}
		pf, err := strconv.ParseFloat(strings.TrimSpace(rec[idx["power_factor"]]), 64)
		if err != nil {
			return nil, fmt.Errorf("parsing power_factor: %w", err)
		}
		out = append(out, models.Reading{
			MeterID:        strings.TrimSpace(rec[idx["meter_id"]]),
			Timestamp:      ts,
			ConsumptionKWh: consumption,
			VoltageV:       voltage,
			CurrentA:       current,
			PowerFactor:    pf,
		})
	}
	return out, nil
}

// ParseEvents reads events.csv:
// meter_id,event_timestamp,event_type,description
func ParseEvents(r io.Reader) ([]models.Event, error) {
	cr := csv.NewReader(r)
	header, err := cr.Read()
	if err != nil {
		return nil, fmt.Errorf("reading header: %w", err)
	}
	idx, err := columnIndex(header, "meter_id", "event_timestamp", "event_type", "description")
	if err != nil {
		return nil, err
	}

	var out []models.Event
	for {
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading row: %w", err)
		}
		ts, err := parseTimestamp(rec[idx["event_timestamp"]])
		if err != nil {
			return nil, err
		}
		out = append(out, models.Event{
			MeterID:        strings.TrimSpace(rec[idx["meter_id"]]),
			EventTimestamp: ts,
			EventType:      strings.TrimSpace(rec[idx["event_type"]]),
			Description:    strings.TrimSpace(rec[idx["description"]]),
		})
	}
	return out, nil
}

func columnIndex(header []string, want ...string) (map[string]int, error) {
	idx := make(map[string]int, len(header))
	for i, h := range header {
		idx[strings.TrimSpace(strings.ToLower(h))] = i
	}
	for _, w := range want {
		if _, ok := idx[w]; !ok {
			return nil, fmt.Errorf("missing expected column %q in header %v", w, header)
		}
	}
	return idx, nil
}
