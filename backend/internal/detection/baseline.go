package detection

import "sort"

// median returns the median of vals. vals is not mutated (a sorted copy is
// used internally).
func median(vals []float64) float64 {
	n := len(vals)
	if n == 0 {
		return 0
	}
	sorted := append([]float64(nil), vals...)
	sort.Float64s(sorted)
	mid := n / 2
	if n%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2
	}
	return sorted[mid]
}

// mad returns the median absolute deviation of vals around the given
// median (not scaled — callers apply the 0.6745 normal-consistency
// constant themselves when computing a modified z-score).
func mad(vals []float64, med float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	devs := make([]float64, len(vals))
	for i, v := range vals {
		d := v - med
		if d < 0 {
			d = -d
		}
		devs[i] = d
	}
	return median(devs)
}

// modifiedZScore is 0.6745*(x-median)/MAD, the standard robust
// alternative to a mean/stddev z-score. Returns 0 when MAD is 0 (a
// perfectly flat series has no meaningful deviation to score).
func modifiedZScore(x, med, madVal float64) float64 {
	if madVal == 0 {
		return 0
	}
	return 0.6745 * (x - med) / madVal
}

// hourlyBaseline holds, for each hour of the day (0-23), the median and
// MAD of consumption_kwh across all days in the dataset for one meter.
type hourlyBaseline struct {
	median [24]float64
	mad    [24]float64
}

func computeHourlyBaseline(consumptionByHour [24][]float64) hourlyBaseline {
	var b hourlyBaseline
	for h := 0; h < 24; h++ {
		m := median(consumptionByHour[h])
		b.median[h] = m
		b.mad[h] = mad(consumptionByHour[h], m)
	}
	return b
}

// ratioBaseline holds the median/MAD of consumption_kwh divided by the
// electrical estimate (voltage*current*power_factor/1000) across all of a
// meter's readings — used for the data-quality (electrical consistency)
// signal.
type ratioBaseline struct {
	median float64
	mad    float64
}

func computeRatioBaseline(ratios []float64) ratioBaseline {
	m := median(ratios)
	return ratioBaseline{median: m, mad: mad(ratios, m)}
}

// electricalEstimateKWh is the consumption implied by V*I*PF, used only as
// a cross-check against the reported consumption_kwh, never as ground
// truth on its own — see the data-quality signal in engine.go.
func electricalEstimateKWh(voltageV, currentA, powerFactor float64) float64 {
	return voltageV * currentA * powerFactor / 1000.0
}
