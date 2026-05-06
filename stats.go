package main

import (
	"math"
	"sort"
)

// sortedCopy returns a sorted copy of the input slice.
func sortedCopy(data []float64) []float64 {
	c := make([]float64, len(data))
	copy(c, data)
	sort.Float64s(c)
	return c
}

// percentile returns the p-th percentile of a sorted slice using linear interpolation.
func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := p / 100.0 * float64(len(sorted)-1)
	lo := int(idx)
	hi := lo + 1
	if hi >= len(sorted) {
		return sorted[len(sorted)-1]
	}
	frac := idx - float64(lo)
	return sorted[lo]*(1-frac) + sorted[hi]*frac
}

func mean(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	s := 0.0
	for _, v := range data {
		s += v
	}
	return s / float64(len(data))
}

func stddev(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	m := mean(data)
	s := 0.0
	for _, v := range data {
		d := v - m
		s += d * d
	}
	return math.Sqrt(s / float64(len(data)))
}

func maxFloat(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	m := data[0]
	for _, v := range data[1:] {
		if v > m {
			m = v
		}
	}
	return m
}

func maxInt(data []int) int {
	if len(data) == 0 {
		return 0
	}
	m := data[0]
	for _, v := range data[1:] {
		if v > m {
			m = v
		}
	}
	return m
}

// intsToFloat converts a slice of ints to a slice of float64.
func intsToFloat(data []int) []float64 {
	out := make([]float64, len(data))
	for i, v := range data {
		out[i] = float64(v)
	}
	return out
}

// pctSet holds the 5th, 50th, and 95th percentile values for a metric.
type pctSet struct{ p5, p50, p95 float64 }

// calcPct returns the 5th, 50th, and 95th percentiles of a float64 slice.
func calcPct(data []float64) pctSet {
	s := sortedCopy(data)
	return pctSet{percentile(s, 5), percentile(s, 50), percentile(s, 95)}
}

// calcPctInt returns the 5th, 50th, and 95th percentiles of an int slice.
func calcPctInt(data []int) pctSet {
	return calcPct(intsToFloat(data))
}
