package main

import (
	"fmt"
	"strings"
)

func (s *Simulator) GenerateReport(res MCResults, timestamp string) string {
	p := s.params
	sb := &strings.Builder{}

	line := func(format string, args ...any) {
		fmt.Fprintf(sb, format+"\n", args...)
	}
	sep := func() { line(strings.Repeat("=", 70)) }

	sep()
	line("# MONTE CARLO SIMULATION REPORT")
	sep()
	line("Date: %s", strings.ReplaceAll(timestamp, "_", " "))
	line("")

	sep()
	line("# SIMULATION PARAMETERS")
	sep()
	line("Initial balance:     $%.2f", p.InitialBalance)
	line("Win rate:            %.1f%%", p.WinRate*100)
	if p.BreakevenPercent > 0 {
		line("Breakeven:           %.1f%%", p.BreakevenPercent*100)
	}
	line("Reward:risk:         %.2f", p.WinMultiplier)
	line("Risk per trade:      %.2f%%", p.RiskPercent*100)
	line("Commission:          %.2f%%", p.Commission*100)
	compStr := "No"
	if p.UseCompounding {
		compStr = "Yes"
	}
	line("Compounding:         %s", compStr)
	line("Trade count:         %d", p.TradeCount)
	line("Simulation count:    %d", p.SimulationCount)
	line("RR model:            %s", p.RRModel)
	switch p.RRModel {
	case "uniform":
		line("RR deviation:        ±%.0f%%", p.RRDeviation*100)
	case "normal":
		line("RR sigma:            %.2f (±%.0f%%)", p.RRSigma, p.RRSigma*100)
	}
	line("")

	sep()
	line("# PERFORMANCE")
	sep()
	line("Elapsed time:  %.2f sec", res.ElapsedTime)
	line("Speed:         %.0f simulations/sec", res.SimsPerSecond)
	line("")

	// Percentile sets for each metric
	type pctSet struct{ p5, p50, p95 float64 }
	calc := func(data []float64) pctSet {
		s := sortedCopy(data)
		return pctSet{percentile(s, 5), percentile(s, 50), percentile(s, 95)}
	}
	calcI := func(data []int) pctSet { return calc(intsToFloat(data)) }

	bal := calc(res.FinalBalances)
	ret := calc(res.ReturnsPercent)
	dd := calc(res.MaxDrawdowns)
	wr := calc(res.WinRates)
	mws := calcI(res.MaxWinStreaks)
	mls := calcI(res.MaxLossStreaks)
	ttr := calcI(res.MaxTradesToRecovery)

	sep()
	line("# RESULTS (5%% / Median / 95%%)")
	sep()

	line("│ Worst (5%%)")
	line("├─ Final balance:           $%.2f", bal.p5)
	line("├─ Return:                  %+.2f%%", ret.p5*100)
	line("├─ Max drawdown:            %.2f%%", dd.p95*100)
	line("├─ Win rate:                %.1f%%", wr.p5*100)
	line("├─ Max win streak:          %.0f", mws.p5)
	line("├─ Max loss streak:         %.0f", mls.p95)
	line("└─ Max TTR:                 %.0f", ttr.p95)

	line("")
	line("│ Median (50%%)")
	line("├─ Final balance:           $%.2f", bal.p50)
	line("├─ Return:                  %+.2f%%", ret.p50*100)
	line("├─ Max drawdown:            %.2f%%", dd.p50*100)
	line("├─ Win rate:                %.1f%%", wr.p50*100)
	line("├─ Max win streak:          %.0f", mws.p50)
	line("├─ Max loss streak:         %.0f", mls.p50)
	line("└─ Max TTR:                 %.0f", ttr.p50)

	line("")
	line("│ Best (95%%)")
	line("├─ Final balance:           $%.2f", bal.p95)
	line("├─ Return:                  %+.2f%%", ret.p95*100)
	line("├─ Max drawdown:            %.2f%%", dd.p5*100)
	line("├─ Win rate:                %.1f%%", wr.p95*100)
	line("├─ Max win streak:          %.0f", mws.p95)
	line("├─ Max loss streak:         %.0f", mls.p5)
	line("└─ Max TTR:                 %.0f", ttr.p5)
	line("")

	sep()
	line("# WORST CASE VALUES")
	sep()
	line("Largest drawdown:           %.2f%%", maxFloat(res.MaxDrawdowns)*100)
	line("Longest loss streak:        %d consecutive losses", maxInt(res.MaxLossStreaks))
	line("Longest TTR:                %d trades to recovery", maxInt(res.MaxTradesToRecovery))
	line("")
	line("[!] Be prepared for these scenarios!")
	line("")

	sep()
	line("# RISK ANALYSIS")
	sep()

	initial := p.InitialBalance
	n := float64(len(res.FinalBalances))
	profitCount, lossCount, ruinCount := 0.0, 0.0, 0.0
	for _, b := range res.FinalBalances {
		if b > initial {
			profitCount++
		} else {
			lossCount++
		}
		if b < initial*0.5 {
			ruinCount++
		}
	}
	line("Probabilities:")
	line("  Profitable simulations:   %.2f%%", profitCount/n*100)
	line("  Losing simulations:       %.2f%%", lossCount/n*100)
	line("  Ruin (>50%% loss):         %.2f%%", ruinCount/n*100)

	// VaR and CVaR calculated on return distribution
	sortedRet := sortedCopy(res.ReturnsPercent)
	var95 := percentile(sortedRet, 5) * 100
	var99 := percentile(sortedRet, 1) * 100
	thresh95 := percentile(sortedRet, 5)
	cvarSum, cvarCount := 0.0, 0.0
	for _, r := range res.ReturnsPercent {
		if r <= thresh95 {
			cvarSum += r
			cvarCount++
		}
	}
	cvar95 := 0.0
	if cvarCount > 0 {
		cvar95 = cvarSum / cvarCount * 100
	}
	m := mean(res.ReturnsPercent)
	std := stddev(res.ReturnsPercent)
	sharpe := 0.0
	if std > 0 {
		sharpe = m / std
	}

	line("")
	line("Value at Risk (95%%):        %.2f%%", var95)
	line("Value at Risk (99%%):        %.2f%%", var99)
	line("CVaR (95%%):                 %.2f%%", cvar95)
	line("Sharpe ratio:               %.2f", sharpe)
	sep()

	return sb.String()
}
