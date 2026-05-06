package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

type StressScenario struct {
	Name    string
	Params  TradingParameters
	Results MCResults
}

// parseStressSteps parses a comma-separated string of stress steps in percent.
// Base scenario (0) is always prepended.
func parseStressSteps(s string) []float64 {
	parts := strings.Split(s, ",")
	steps := []float64{0}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if v, err := strconv.ParseFloat(p, 64); err == nil && v > 0 {
			steps = append(steps, -v/100)
		}
	}
	return steps
}

// RunStressTest runs simulations with progressively worse parameters.
func RunStressTest(params TradingParameters, steps []float64) []StressScenario {
	scenarios := make([]StressScenario, len(steps))

	for i, step := range steps {
		p := params
		if step != 0 {
			p.WinRate = math.Max(0, params.WinRate*(1+step))
			p.WinMultiplier = math.Max(0, params.WinMultiplier*(1+step))
		}

		name := "Base"
		if step != 0 {
			name = fmt.Sprintf("Stress %+.0f%%", step*100)
		}

		sim := NewSimulator(p)
		scenarios[i] = StressScenario{
			Name:    name,
			Params:  p,
			Results: sim.RunMonteCarlo(),
		}
	}
	return scenarios
}

// GenerateStressReport generates a stress test report for all scenarios.
func GenerateStressReport(scenarios []StressScenario) string {
	sb := &strings.Builder{}

	line := func(format string, args ...any) {
		fmt.Fprintf(sb, format+"\n", args...)
	}
	sep := func() { line(strings.Repeat("=", 70)) }

	sep()
	line("# STRESS TEST")
	sep()

	for _, sc := range scenarios {
		p := sc.Params
		res := sc.Results
		n := len(res.FinalBalances)

		bal := calcPct(res.FinalBalances)
		ret := calcPct(res.ReturnsPercent)
		dd := calcPct(res.MaxDrawdowns)
		mls := calcPctInt(res.MaxLossStreaks)
		ttr := calcPctInt(res.MaxTradesToRecovery)

		profitCount := 0.0
		for _, b := range res.FinalBalances {
			if b > p.InitialBalance {
				profitCount++
			}
		}

		line("│ %s (Win rate %.1f%% / RR %.2f)",
			sc.Name, p.WinRate*100, p.WinMultiplier)
		line("├─ Final balance:            $%.0f / $%.0f / $%.0f",
			bal.p5, bal.p50, bal.p95)
		line("├─ Return:                   %+.2f%% / %+.2f%% / %+.2f%%",
			ret.p5*100, ret.p50*100, ret.p95*100)
		line("├─ Max drawdown:             %.2f%% / %.2f%% / %.2f%%",
			dd.p95*100, dd.p50*100, dd.p5*100)
		line("├─ Max loss streak:          %.0f / %.0f / %.0f",
			mls.p95, mls.p50, mls.p5)
		line("├─ Max TTR:                  %.0f / %.0f / %.0f",
			ttr.p95, ttr.p50, ttr.p5)
		line("├─ Largest drawdown:         %.2f%%", maxFloat(res.MaxDrawdowns)*100)
		line("├─ Longest loss streak:      %d", maxInt(res.MaxLossStreaks))
		line("├─ Longest TTR:              %d", maxInt(res.MaxTradesToRecovery))
		line("└─ Profitable simulations:   %.2f%%", profitCount/float64(n)*100)
		line("")
	}

	sep()
	return sb.String()
}
