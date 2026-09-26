package seed

import (
	"strings"
	"testing"
	"time"
)

func TestParseReadingsHappyPath(t *testing.T) {
	csv := "meter_id,timestamp,consumption_kwh,voltage_v,current_a,power_factor,status\n" +
		"M-101,2026-01-01 00:00:00,12.5,220.1,5.2,0.95,OK\n" +
		" M-102 ,2026-01-01 01:00, 8.3 , 219.9 , 4.1 , 0.94 ,OK\n"

	readings, err := ParseReadings(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("ParseReadings: %v", err)
	}
	if len(readings) != 2 {
		t.Fatalf("got %d readings, want 2", len(readings))
	}

	r0 := readings[0]
	if r0.MeterID != "M-101" || r0.ConsumptionKWh != 12.5 || r0.VoltageV != 220.1 || r0.CurrentA != 5.2 || r0.PowerFactor != 0.95 {
		t.Errorf("unexpected first reading: %+v", r0)
	}
	wantTS := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if !r0.Timestamp.Equal(wantTS) {
		t.Errorf("Timestamp = %v, want %v", r0.Timestamp, wantTS)
	}

	// Second row exercises: whitespace-trimmed fields and the
	// no-seconds timestamp layout ("2006-01-02 15:04").
	r1 := readings[1]
	if r1.MeterID != "M-102" {
		t.Errorf("MeterID = %q, want trimmed M-102", r1.MeterID)
	}
	wantTS1 := time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC)
	if !r1.Timestamp.Equal(wantTS1) {
		t.Errorf("Timestamp = %v, want %v", r1.Timestamp, wantTS1)
	}
}

func TestParseReadingsMissingColumn(t *testing.T) {
	csv := "meter_id,timestamp,consumption_kwh,voltage_v,current_a\n" + // missing power_factor
		"M-101,2026-01-01 00:00:00,12.5,220.1,5.2\n"
	if _, err := ParseReadings(strings.NewReader(csv)); err == nil {
		t.Fatal("expected an error for a missing required column")
	}
}

func TestParseReadingsBadTimestamp(t *testing.T) {
	csv := "meter_id,timestamp,consumption_kwh,voltage_v,current_a,power_factor\n" +
		"M-101,not-a-timestamp,12.5,220.1,5.2,0.95\n"
	if _, err := ParseReadings(strings.NewReader(csv)); err == nil {
		t.Fatal("expected an error for an unparseable timestamp")
	}
}

func TestParseReadingsBadNumbers(t *testing.T) {
	header := "meter_id,timestamp,consumption_kwh,voltage_v,current_a,power_factor\n"
	cases := map[string]string{
		"consumption_kwh": header + "M-101,2026-01-01 00:00:00,oops,220.1,5.2,0.95\n",
		"voltage_v":       header + "M-101,2026-01-01 00:00:00,12.5,oops,5.2,0.95\n",
		"current_a":       header + "M-101,2026-01-01 00:00:00,12.5,220.1,oops,0.95\n",
		"power_factor":    header + "M-101,2026-01-01 00:00:00,12.5,220.1,5.2,oops\n",
	}
	for field, csv := range cases {
		t.Run(field, func(t *testing.T) {
			if _, err := ParseReadings(strings.NewReader(csv)); err == nil {
				t.Fatalf("expected an error for an unparseable %s", field)
			}
		})
	}
}

func TestParseReadingsEmptyHeaderFails(t *testing.T) {
	if _, err := ParseReadings(strings.NewReader("")); err == nil {
		t.Fatal("expected an error reading the header from an empty file")
	}
}

func TestParseReadingsMalformedRow(t *testing.T) {
	csv := "meter_id,timestamp,consumption_kwh,voltage_v,current_a,power_factor\n" +
		"\"unterminated,2026-01-01 00:00:00,12.5,220.1,5.2,0.95\n"
	if _, err := ParseReadings(strings.NewReader(csv)); err == nil {
		t.Fatal("expected an error for a malformed CSV row")
	}
}

func TestParseEventsHappyPath(t *testing.T) {
	csv := "meter_id,event_timestamp,event_type,description\n" +
		"M-104,2026-01-02 08:00:00,OPERATIONAL_CHANGE, Cambio de turno \n"

	events, err := ParseEvents(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("ParseEvents: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	e := events[0]
	if e.MeterID != "M-104" || e.EventType != "OPERATIONAL_CHANGE" || e.Description != "Cambio de turno" {
		t.Errorf("unexpected event: %+v", e)
	}
	wantTS := time.Date(2026, 1, 2, 8, 0, 0, 0, time.UTC)
	if !e.EventTimestamp.Equal(wantTS) {
		t.Errorf("EventTimestamp = %v, want %v", e.EventTimestamp, wantTS)
	}
}

func TestParseEventsMissingColumn(t *testing.T) {
	csv := "meter_id,event_timestamp,event_type\n" + // missing description
		"M-104,2026-01-02 08:00:00,OPERATIONAL_CHANGE\n"
	if _, err := ParseEvents(strings.NewReader(csv)); err == nil {
		t.Fatal("expected an error for a missing required column")
	}
}

func TestParseEventsBadTimestamp(t *testing.T) {
	csv := "meter_id,event_timestamp,event_type,description\n" +
		"M-104,not-a-timestamp,OPERATIONAL_CHANGE,desc\n"
	if _, err := ParseEvents(strings.NewReader(csv)); err == nil {
		t.Fatal("expected an error for an unparseable timestamp")
	}
}

func TestParseEventsEmptyHeaderFails(t *testing.T) {
	if _, err := ParseEvents(strings.NewReader("")); err == nil {
		t.Fatal("expected an error reading the header from an empty file")
	}
}

func TestParseEventsMalformedRow(t *testing.T) {
	csv := "meter_id,event_timestamp,event_type,description\n" +
		"\"unterminated,2026-01-02 08:00:00,OPERATIONAL_CHANGE,desc\n"
	if _, err := ParseEvents(strings.NewReader(csv)); err == nil {
		t.Fatal("expected an error for a malformed CSV row")
	}
}
