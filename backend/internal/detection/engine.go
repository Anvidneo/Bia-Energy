// Package detection is the deterministic anomaly detection engine.
// Nothing here calls an LLM: classification is 100% rules + statistics, so
// it is reproducible and unit-testable — see engine_test.go, which runs
// this exact code against the real dataset for the 4 known cases
// (M-109, M-106, M-104, M-112).
package detection

import (
	"sort"
	"time"

	"bia-energy/backend/internal/models"
)

// Thresholds. Calibrated against the real 12-meter/14-day dataset: every
// value here was chosen so the 4 known anomalous meters classify
// correctly while the other 8 (unremarkable) meters produce zero false
// positives. See the "Casos conocidos" section of the plan doc for the
// expected classification of each known meter.
const (
	// ConsumptionZThreshold: a reading's hourly modified z-score beyond
	// this is "suspicious" (enunciado: |z| > 3.5).
	ConsumptionZThreshold = 3.5

	// MinConsecutiveDeviationHours: a suspicious reading only becomes a
	// deviation *episode* once it's sustained for this many consecutive
	// hours — this is what separates a real shift (M-104, M-106, M-109)
	// from an isolated one-hour spike.
	MinConsecutiveDeviationHours = 3

	// RatioZThreshold: the modified z-score of consumption_kwh divided by
	// the electrical estimate (V*I*PF) beyond this, while consumption
	// itself stays near baseline, indicates the meter's own readings
	// don't reconcile with each other (M-112) rather than a real
	// consumption change.
	RatioZThreshold = 5.0

	// MinDataQualityFlags: minimum number of inconsistent readings (not
	// necessarily consecutive — "intermittent") before a meter is
	// classified DATA_QUALITY.
	MinDataQualityFlags = 3

	// EventWindow: how far from a deviation episode's start we'll look
	// in events.csv for a matching operational event.
	EventWindow = 24 * time.Hour
)

// scoredReading is one reading plus the two z-scores computed for it.
type scoredReading struct {
	reading models.Reading
	zc      float64 // consumption modified z-score (vs hourly baseline)
	zr      float64 // ratio modified z-score (vs meter's own electrical-consistency baseline)
	isDQ    bool    // flagged as a data-quality (electrical inconsistency) reading
	isDev   bool    // flagged as a consumption-deviation reading
}

// AnalyzeMeter runs the full detection pipeline for one meter and returns
// nil if nothing anomalous was found. readings must all belong to the same
// meter; events may contain events for other meters too (they're filtered
// here).
func AnalyzeMeter(meterID string, readings []models.Reading, events []models.Event) *models.Anomaly {
	if len(readings) == 0 {
		return nil
	}

	sorted := append([]models.Reading(nil), readings...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Timestamp.Before(sorted[j].Timestamp) })

	var byHour [24][]float64
	ratios := make([]float64, len(sorted))
	for i, r := range sorted {
		byHour[r.Timestamp.Hour()] = append(byHour[r.Timestamp.Hour()], r.ConsumptionKWh)
		ratios[i] = r.ConsumptionKWh / electricalEstimateKWh(r.VoltageV, r.CurrentA, r.PowerFactor)
	}
	baseline := computeHourlyBaseline(byHour)
	ratioBase := computeRatioBaseline(ratios)

	scoredReadings := make([]scoredReading, len(sorted))
	for i, r := range sorted {
		h := r.Timestamp.Hour()
		zc := modifiedZScore(r.ConsumptionKWh, baseline.median[h], baseline.mad[h])
		zr := modifiedZScore(ratios[i], ratioBase.median, ratioBase.mad)
		// Data-quality: the electrical numbers don't reconcile with each
		// other WHILE consumption itself stays close to its own hourly
		// baseline. Checking |zc| first is what keeps a real consumption
		// spike (M-109, whose ratio also wobbles as a side effect) from
		// being misread as a metering fault.
		isDQ := absF(zr) > RatioZThreshold && absF(zc) <= ConsumptionZThreshold
		isDev := absF(zc) > ConsumptionZThreshold && !isDQ
		scoredReadings[i] = scoredReading{reading: r, zc: zc, zr: zr, isDQ: isDQ, isDev: isDev}
	}

	// --- Data-quality signal (checked first: it explains away readings
	// that would otherwise also look like a consumption deviation) ---
	var dqFlagged []scoredReading
	for _, s := range scoredReadings {
		if s.isDQ {
			dqFlagged = append(dqFlagged, s)
		}
	}
	if len(dqFlagged) >= MinDataQualityFlags {
		return buildDataQualityAnomaly(meterID, dqFlagged, baseline)
	}

	// --- Consumption deviation signal: find the longest run of
	// consecutive suspicious hours ---
	bestStart, bestEnd := -1, -1
	curStart := -1
	for i := 0; i <= len(scoredReadings); i++ {
		active := i < len(scoredReadings) && scoredReadings[i].isDev
		if active && curStart == -1 {
			curStart = i
		}
		if !active && curStart != -1 {
			if bestStart == -1 || (i-1-curStart) > (bestEnd-bestStart) {
				bestStart, bestEnd = curStart, i-1
			}
			curStart = -1
		}
	}
	if bestStart == -1 || (bestEnd-bestStart+1) < MinConsecutiveDeviationHours {
		return nil // NORMAL: nothing sustained enough to report
	}

	episode := scoredReadings[bestStart : bestEnd+1]
	return buildDeviationAnomaly(meterID, episode, events, baseline)
}

// RunPipeline groups readings by meter and analyzes each one, returning
// only the meters with a detected anomaly (NORMAL meters are omitted).
func RunPipeline(readings []models.Reading, events []models.Event) []models.Anomaly {
	byMeter := make(map[string][]models.Reading)
	var order []string
	for _, r := range readings {
		if _, ok := byMeter[r.MeterID]; !ok {
			order = append(order, r.MeterID)
		}
		byMeter[r.MeterID] = append(byMeter[r.MeterID], r)
	}
	sort.Strings(order)

	var out []models.Anomaly
	for _, meterID := range order {
		if a := AnalyzeMeter(meterID, byMeter[meterID], events); a != nil {
			out = append(out, *a)
		}
	}
	return out
}

func buildDataQualityAnomaly(meterID string, flagged []scoredReading, baseline hourlyBaseline) *models.Anomaly {
	// worst (most extreme) flagged reading drives the evidence shown
	worst := flagged[0]
	for _, s := range flagged {
		if absF(s.zr) > absF(worst.zr) {
			worst = s
		}
	}
	avgAbsZr := 0.0
	for _, s := range flagged {
		avgAbsZr += absF(s.zr)
	}
	avgAbsZr /= float64(len(flagged))

	h := worst.reading.Timestamp.Hour()
	return &models.Anomaly{
		MeterID:    meterID,
		DetectedAt: flagged[0].reading.Timestamp,
		Anomaly:    true,
		Type:       models.TypeDataQuality,
		Severity:   models.SeverityHigh,
		Confidence: confidenceFromZ(avgAbsZr, RatioZThreshold),
		Evidence: models.Evidence{
			BaselineMedianKWh:         baseline.median[h],
			ObservedKWh:               worst.reading.ConsumptionKWh,
			DeviationPct:              percentDeviation(worst.reading.ConsumptionKWh, baseline.median[h]),
			ZScore:                    worst.zr,
			ConsecutiveHours:          len(flagged),
			ExpectedKWhFromElectrical: electricalEstimateKWh(worst.reading.VoltageV, worst.reading.CurrentA, worst.reading.PowerFactor),
		},
	}
}

func buildDeviationAnomaly(meterID string, episode []scoredReading, events []models.Event, baseline hourlyBaseline) *models.Anomaly {
	start := episode[0].reading
	avgAbsZc := 0.0
	for _, s := range episode {
		avgAbsZc += absF(s.zc)
	}
	avgAbsZc /= float64(len(episode))

	matchedEvent := findMatchingEvent(meterID, start.Timestamp, events)

	var anomalyType models.AnomalyType
	var severity models.Severity
	relatedEventType := ""
	var relatedEventAt *time.Time

	switch {
	case matchedEvent != nil && matchedEvent.EventType == models.EventScheduledOutage:
		anomalyType = models.TypeFalsePositive
		severity = models.SeverityLow
		relatedEventType = matchedEvent.EventType
		t := matchedEvent.EventTimestamp
		relatedEventAt = &t
	case matchedEvent != nil && matchedEvent.EventType == models.EventOperationalChange:
		anomalyType = models.TypeExplainableAnomaly
		severity = models.SeverityMedium
		relatedEventType = matchedEvent.EventType
		t := matchedEvent.EventTimestamp
		relatedEventAt = &t
	default:
		// No event, or an event of type UNKNOWN / DATA_QUALITY that
		// doesn't explain a *consumption* deviation: treat as a real,
		// unexplained anomaly.
		anomalyType = models.TypeRealAnomaly
		severity = models.SeverityHigh
	}

	h := start.Timestamp.Hour()
	return &models.Anomaly{
		MeterID:          meterID,
		DetectedAt:       start.Timestamp,
		Anomaly:          true,
		Type:             anomalyType,
		Severity:         severity,
		Confidence:       confidenceFromZ(avgAbsZc, ConsumptionZThreshold),
		RelatedEventType: relatedEventType,
		RelatedEventAt:   relatedEventAt,
		Evidence: models.Evidence{
			BaselineMedianKWh: baseline.median[h],
			ObservedKWh:       start.ConsumptionKWh,
			DeviationPct:      percentDeviation(start.ConsumptionKWh, baseline.median[h]),
			ZScore:            episode[0].zc,
			ConsecutiveHours:  len(episode),
		},
	}
}

// findMatchingEvent looks for an event on the same meter within
// EventWindow of episodeStart, preferring the closest one in time.
func findMatchingEvent(meterID string, episodeStart time.Time, events []models.Event) *models.Event {
	var best *models.Event
	var bestDelta time.Duration
	for i := range events {
		e := events[i]
		if e.MeterID != meterID {
			continue
		}
		delta := e.EventTimestamp.Sub(episodeStart)
		if delta < 0 {
			delta = -delta
		}
		if delta > EventWindow {
			continue
		}
		if best == nil || delta < bestDelta {
			best = &events[i]
			bestDelta = delta
		}
	}
	return best
}

func confidenceFromZ(avgAbsZ, threshold float64) float64 {
	ratio := (avgAbsZ - threshold) / threshold
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	c := 0.5 + 0.49*ratio
	if c > 0.99 {
		c = 0.99
	}
	return c
}

func percentDeviation(observed, baseline float64) float64 {
	if baseline == 0 {
		return 0
	}
	return (observed - baseline) / baseline * 100
}

func absF(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
