package main

import (
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
)

func SaveCSV(res MCResults, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", path, err)
	}
	defer f.Close()

	// Write UTF-8 BOM for Excel compatibility
	if _, err := f.WriteString("\xef\xbb\xbf"); err != nil {
		return fmt.Errorf("failed to write BOM: %w", err)
	}

	w := csv.NewWriter(f)
	w.Comma = ';'

	headers := []string{
		"Simulation", "Final_Balance", "Return_%", "Max_Drawdown_%",
		"Win_Rate_%", "Winning_Trades", "Max_Win_Streak", "Max_Loss_Streak", "Max_TTR",
	}
	if err := w.Write(headers); err != nil {
		return fmt.Errorf("failed to write CSV headers: %w", err)
	}

	// Use comma as decimal separator for Excel compatibility
	ff := func(v float64) string {
		return strings.ReplaceAll(strconv.FormatFloat(v, 'f', 4, 64), ".", ",")
	}

	for i := range res.FinalBalances {
		row := []string{
			strconv.Itoa(i + 1),
			ff(res.FinalBalances[i]),
			ff(res.ReturnsPercent[i] * 100),
			ff(res.MaxDrawdowns[i] * 100),
			ff(res.WinRates[i] * 100),
			strconv.Itoa(res.WinningTrades[i]),
			strconv.Itoa(res.MaxWinStreaks[i]),
			strconv.Itoa(res.MaxLossStreaks[i]),
			strconv.Itoa(res.MaxTradesToRecovery[i]),
		}
		if err := w.Write(row); err != nil {
			return fmt.Errorf("failed to write row %d: %w", i+1, err)
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return fmt.Errorf("failed to flush CSV buffer: %w", err)
	}

	return nil
}

// SaveSVG saves an SVG chart of equity curves to the given path.
func SaveSVG(res MCResults, params TradingParameters, path string) error {
	const W, H = 1100, 620
	const padL, padR, padT, padB = 90, 40, 30, 50

	plotW := W - padL - padR
	plotH := H - padT - padB

	bgColors := []string{
		"#4a9eff", "#ff6b6b", "#6bffb8", "#ffd36b",
		"#b86bff", "#ff9f6b", "#6bd5ff", "#ff6bd5",
		"#9fff6b", "#ff6b9f", "#ffec6b", "#6bffe0",
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	maxCurves := params.SVGMaxCurves
	if maxCurves > params.SimulationCount {
		maxCurves = params.SimulationCount
	}

	// Ruin threshold: with compounding balance never reaches exact 0 in float64,
	// so we treat a drop below 0.01% of initial balance as ruin.
	ruinThreshold := params.InitialBalance * 0.0001

	// firstRuinTrade returns the trade index where balance first dropped below
	// the ruin threshold. Returns len(curve) if ruin never occurred.
	firstRuinTrade := func(curve []float64) int {
		for i, v := range curve {
			if v <= ruinThreshold {
				return i
			}
		}
		return len(curve)
	}

	// curveArea returns the sum of all balance values (area under the curve).
	// Used as a tiebreaker when two curves have the same final balance:
	// larger area = stayed higher longer = better trajectory.
	curveArea := func(curve []float64) float64 {
		s := 0.0
		for _, v := range curve {
			s += v
		}
		return s
	}

	// Sort by final balance to match report percentiles.
	// Tiebreaker 1: curve area. Tiebreaker 2: firstRuinTrade.
	n := len(res.FinalBalances)
	sortedIdx := make([]int, n)
	for i := range sortedIdx {
		sortedIdx[i] = i
	}
	sort.Slice(sortedIdx, func(a, b int) bool {
		ia := sortedIdx[a]
		ib := sortedIdx[b]
		fa := res.FinalBalances[ia]
		fb := res.FinalBalances[ib]
		if math.Abs(fa-fb) > 1e-9 {
			return fa < fb
		}
		aa := curveArea(res.EquityCurves[ia])
		ab := curveArea(res.EquityCurves[ib])
		if math.Abs(aa-ab) > 1e-9 {
			return aa < ab
		}
		return firstRuinTrade(res.EquityCurves[ia]) < firstRuinTrade(res.EquityCurves[ib])
	})
	idx5 := sortedIdx[int(0.05*float64(n-1))]
	idx50 := sortedIdx[int(0.50*float64(n-1))]
	idx95 := sortedIdx[int(0.95*float64(n-1))]

	// Find min/max balance across all displayed and percentile curves.
	// Only positive values are considered (balance >= 0).
	minB, maxB := math.MaxFloat64, -math.MaxFloat64
	for i := 0; i < maxCurves; i++ {
		idx := sortedIdx[(i*n+n/2)/maxCurves]
		for _, v := range res.EquityCurves[idx] {
			if v > 0 && v < minB {
				minB = v
			}
			if v > maxB {
				maxB = v
			}
		}
	}
	for _, idx := range []int{idx5, idx50, idx95} {
		for _, v := range res.EquityCurves[idx] {
			if v > 0 && v < minB {
				minB = v
			}
			if v > maxB {
				maxB = v
			}
		}
	}
	if minB == math.MaxFloat64 {
		minB = 1
	}
	if maxB <= 0 {
		maxB = params.InitialBalance
	}

	tradeCount := params.TradeCount

	toX := func(trade int) float64 {
		return float64(padL) + float64(trade)/float64(tradeCount)*float64(plotW)
	}

	// Use log scale with compounding — curves don't cluster at the bottom
	// and a 100x range reads evenly. Use linear scale for fixed position size.
	useLogScale := params.UseCompounding && maxB/minB > 5

	var toY func(bal float64) float64
	var yAxisVals []float64

	if useLogScale {
		logMin := math.Log10(math.Max(minB*0.7, 1))
		logMax := math.Log10(maxB * 1.3)
		logRange := logMax - logMin
		if logRange == 0 {
			logRange = 1
		}
		toY = func(bal float64) float64 {
			if bal <= 0 {
				return float64(padT + plotH)
			}
			frac := (math.Log10(bal) - logMin) / logRange
			if frac < 0 {
				frac = 0
			}
			if frac > 1 {
				frac = 1
			}
			return float64(padT+plotH) - frac*float64(plotH)
		}
		// Y axis labels: powers of 10 with 2x and 5x intermediates
		for exp := int(math.Floor(logMin)); exp <= int(math.Ceil(logMax)); exp++ {
			v := math.Pow(10, float64(exp))
			if v >= math.Pow(10, logMin)*0.9 && v <= math.Pow(10, logMax)*1.1 {
				yAxisVals = append(yAxisVals, v)
			}
			for _, mult := range []float64{2, 5} {
				v2 := math.Pow(10, float64(exp)) * mult
				if v2 >= math.Pow(10, logMin)*0.9 && v2 <= math.Pow(10, logMax)*1.1 {
					yAxisVals = append(yAxisVals, v2)
				}
			}
		}
	} else {
		rangeB := maxB - minB
		if rangeB == 0 {
			rangeB = 1
		}
		pad := rangeB * 0.05
		dispMin := math.Max(0, minB-pad)
		dispMax := maxB + pad
		dispRange := dispMax - dispMin
		toY = func(bal float64) float64 {
			frac := (bal - dispMin) / dispRange
			if frac < 0 {
				frac = 0
			}
			if frac > 1 {
				frac = 1
			}
			return float64(padT+plotH) - frac*float64(plotH)
		}
		for i := 0; i <= 6; i++ {
			yAxisVals = append(yAxisVals, dispMin+float64(i)/6*dispRange)
		}
	}

	// trimCurve truncates the curve at the first ruin point (inclusive).
	trimCurve := func(curve []float64) []float64 {
		for i, v := range curve {
			if v <= ruinThreshold {
				return curve[:i+1]
			}
		}
		return curve
	}

	polyline := func(curve []float64, color, width, opacity string) string {
		curve = trimCurve(curve)
		pts := make([]string, len(curve))
		for j, v := range curve {
			pts[j] = fmt.Sprintf("%.1f,%.1f", toX(j), toY(v))
		}
		return fmt.Sprintf(`<polyline points="%s" fill="none" stroke="%s" stroke-width="%s" opacity="%s"/>`,
			strings.Join(pts, " "), color, width, opacity)
	}

	fmt.Fprintf(f, `<svg width="%d" height="%d" xmlns="http://www.w3.org/2000/svg" font-family="Arial,sans-serif">`, W, H)
	fmt.Fprintf(f, `<rect width="%d" height="%d" fill="#0f1117"/>`, W, H)

	gridColor := "#1e2030"

	// Collect percentile Y positions to avoid overlapping grid labels
	percentileYPos := []float64{
		toY(res.FinalBalances[idx5]),
		toY(res.FinalBalances[idx50]),
		toY(res.FinalBalances[idx95]),
	}

	// Horizontal grid lines and Y axis labels
	for _, bal := range yAxisVals {
		y := toY(bal)

		// Skip label if too close to a percentile label
		tooClose := false
		for _, py := range percentileYPos {
			if math.Abs(y-py) < 14 {
				tooClose = true
				break
			}
		}
		fmt.Fprintf(f, `<line x1="%d" y1="%.1f" x2="%d" y2="%.1f" stroke="%s" stroke-width="1"/>`,
			padL, y, padL+plotW, y, gridColor)
		if tooClose {
			continue
		}

		label := fmt.Sprintf("$%.0f", bal)
		fmt.Fprintf(f, `<text x="%d" y="%.1f" fill="#5a5a7a" font-size="11" text-anchor="end" dominant-baseline="middle">%s</text>`,
			padL-8, y, label)
	}
	if useLogScale {
		fmt.Fprintf(f, `<text x="%d" y="%d" fill="#5a5a7a" font-size="10" text-anchor="end">log</text>`,
			padL-2, padT+12)
	}

	// Vertical grid lines and X axis labels
	xSteps := 10
	for i := 0; i <= xSteps; i++ {
		trade := int(float64(tradeCount) * float64(i) / float64(xSteps))
		x := toX(trade)
		fmt.Fprintf(f, `<line x1="%.1f" y1="%d" x2="%.1f" y2="%d" stroke="%s" stroke-width="1"/>`,
			x, padT, x, padT+plotH, gridColor)
		fmt.Fprintf(f, `<text x="%.1f" y="%d" fill="#5a5a7a" font-size="11" text-anchor="middle">%d</text>`,
			x, padT+plotH+18, trade)
	}

	// Plot border
	fmt.Fprintf(f, `<rect x="%d" y="%d" width="%d" height="%d" fill="none" stroke="#2a2a4a" stroke-width="1"/>`,
		padL, padT, plotW, plotH)

	// Clip path to keep curves inside the plot area
	fmt.Fprintf(f, `<defs><clipPath id="plot"><rect x="%d" y="%d" width="%d" height="%d"/></clipPath></defs>`,
		padL, padT, plotW, plotH)
	fmt.Fprintf(f, `<g clip-path="url(#plot)">`)

	// Background curves — evenly sampled across the sorted distribution
	for i := 0; i < maxCurves; i++ {
		idx := sortedIdx[(i*n+n/2)/maxCurves]
		color := bgColors[idx%len(bgColors)]
		fmt.Fprintln(f, polyline(res.EquityCurves[idx], color, "0.6", "0.15"))
	}

	// Dashed line at initial balance
	y0 := toY(params.InitialBalance)
	fmt.Fprintf(f, `<line x1="%d" y1="%.1f" x2="%d" y2="%.1f" stroke="#ffffff" stroke-width="1" stroke-dasharray="5,5" opacity="0.25"/>`,
		padL, y0, padL+plotW, y0)

	// Percentile curves drawn in order: best, median, worst
	// (worst on top as the most critical risk reference)
	fmt.Fprintln(f, polyline(res.EquityCurves[idx95], "#44dd77", "2", "1"))
	fmt.Fprintln(f, polyline(res.EquityCurves[idx50], "#ffcc00", "2", "1"))
	fmt.Fprintln(f, polyline(res.EquityCurves[idx5], "#ff4444", "2", "1"))

	fmt.Fprintf(f, `</g>`)

	// Axis labels
	cx := float64(padL) + float64(plotW)/2
	fmt.Fprintf(f, `<text x="%.1f" y="%d" fill="#8888aa" font-size="12" text-anchor="middle">Trade number</text>`,
		cx, H-8)
	fmt.Fprintf(f, `<text x="0" y="0" fill="#8888aa" font-size="12" text-anchor="middle" transform="rotate(-90) translate(-%d,%d)">Balance ($)</text>`,
		padT+plotH/2, 16)

	// Percentile labels on Y axis with dashed reference lines
	type yLabel struct {
		idx   int
		color string
		label string
	}
	yLabels := []yLabel{
		{idx95, "#44dd77", fmt.Sprintf("$%.0f", res.FinalBalances[idx95])},
		{idx50, "#ffcc00", fmt.Sprintf("$%.0f", res.FinalBalances[idx50])},
		{idx5, "#ff4444", fmt.Sprintf("$%.0f", res.FinalBalances[idx5])},
	}

	for _, lbl := range yLabels {
		curve := res.EquityCurves[lbl.idx]
		finalBalance := curve[len(curve)-1]
		yPos := toY(finalBalance)
		xEnd := float64(padL + plotW)

		// Dashed horizontal line from end of curve to Y axis
		fmt.Fprintf(f, `<line x1="%.1f" y1="%.1f" x2="%d" y2="%.1f" stroke="%s" stroke-width="1" stroke-dasharray="3,4" opacity="0.6"/>`,
			xEnd, yPos, padL, yPos, lbl.color)

		// Label on Y axis
		fmt.Fprintf(f, `<text x="%d" y="%.1f" fill="%s" font-size="11" text-anchor="end" dominant-baseline="middle">%s</text>`,
			padL-8, yPos, lbl.color, lbl.label)
	}

	fmt.Fprintf(f, `</svg>`)
	return nil
}
