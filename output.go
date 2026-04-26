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

	ruinThreshold := params.InitialBalance * 0.0001

	firstRuinTrade := func(curve []float64) int {
		for i, v := range curve {
			if v <= ruinThreshold {
				return i
			}
		}
		return len(curve)
	}

	curveArea := func(curve []float64) float64 {
		s := 0.0
		for _, v := range curve {
			s += v
		}
		return s
	}

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

	minB, maxB := math.MaxFloat64, -math.MaxFloat64
	for i := 0; i < maxCurves; i++ {
		idx := i * params.SimulationCount / maxCurves
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

	for _, bal := range yAxisVals {
		y := toY(bal)
		fmt.Fprintf(f, `<line x1="%d" y1="%.1f" x2="%d" y2="%.1f" stroke="%s" stroke-width="1"/>`,
			padL, y, padL+plotW, y, gridColor)
		label := fmt.Sprintf("$%.0f", bal)
		fmt.Fprintf(f, `<text x="%d" y="%.1f" fill="#5a5a7a" font-size="11" text-anchor="end" dominant-baseline="middle">%s</text>`,
			padL-8, y, label)
	}
	if useLogScale {
		fmt.Fprintf(f, `<text x="%d" y="%d" fill="#5a5a7a" font-size="10" text-anchor="end">log</text>`,
			padL-2, padT+12)
	}

	xSteps := 10
	for i := 0; i <= xSteps; i++ {
		trade := int(float64(tradeCount) * float64(i) / float64(xSteps))
		x := toX(trade)
		fmt.Fprintf(f, `<line x1="%.1f" y1="%d" x2="%.1f" y2="%d" stroke="%s" stroke-width="1"/>`,
			x, padT, x, padT+plotH, gridColor)
		fmt.Fprintf(f, `<text x="%.1f" y="%d" fill="#5a5a7a" font-size="11" text-anchor="middle">%d</text>`,
			x, padT+plotH+18, trade)
	}

	fmt.Fprintf(f, `<rect x="%d" y="%d" width="%d" height="%d" fill="none" stroke="#2a2a4a" stroke-width="1"/>`,
		padL, padT, plotW, plotH)

	fmt.Fprintf(f, `<defs><clipPath id="plot"><rect x="%d" y="%d" width="%d" height="%d"/></clipPath></defs>`,
		padL, padT, plotW, plotH)
	fmt.Fprintf(f, `<g clip-path="url(#plot)">`)

	for i := 0; i < maxCurves; i++ {
		idx := i * params.SimulationCount / maxCurves
		color := bgColors[idx%len(bgColors)]
		fmt.Fprintln(f, polyline(res.EquityCurves[idx], color, "0.6", "0.15"))
	}

	y0 := toY(params.InitialBalance)
	fmt.Fprintf(f, `<line x1="%d" y1="%.1f" x2="%d" y2="%.1f" stroke="#ffffff" stroke-width="1" stroke-dasharray="5,5" opacity="0.25"/>`,
		padL, y0, padL+plotW, y0)

	fmt.Fprintln(f, polyline(res.EquityCurves[idx50], "#ffcc00", "2", "0.9"))
	fmt.Fprintln(f, polyline(res.EquityCurves[idx95], "#44dd77", "2.5", "1"))
	fmt.Fprintln(f, polyline(res.EquityCurves[idx5], "#ff4444", "2.5", "1"))

	fmt.Fprintf(f, `</g>`)

	cx := float64(padL) + float64(plotW)/2
	fmt.Fprintf(f, `<text x="%.1f" y="%d" fill="#8888aa" font-size="12" text-anchor="middle">Trade number</text>`,
		cx, H-8)
	fmt.Fprintf(f, `<text x="0" y="0" fill="#8888aa" font-size="12" text-anchor="middle" transform="rotate(-90) translate(-%d,%d)">Balance ($)</text>`,
		padT+plotH/2, 16)

	legendX := padL + 12
	legendY := padT + 15
	items := [][2]string{
		{"#44dd77", fmt.Sprintf("Best 95%%   ($%.0f)", res.FinalBalances[idx95])},
		{"#ffcc00", fmt.Sprintf("Median      ($%.0f)", res.FinalBalances[idx50])},
		{"#ff4444", fmt.Sprintf("Worst 5%%   ($%.0f)", res.FinalBalances[idx5])},
	}
	fmt.Fprintf(f, `<rect x="%d" y="%d" width="205" height="72" fill="#0a0c12" fill-opacity="0.85" rx="5" stroke="#2a2a4a" stroke-width="1"/>`,
		legendX-8, legendY-8)
	for i, item := range items {
		ly := legendY + i*22
		fmt.Fprintf(f, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="%s" stroke-width="2.5"/>`,
			legendX, ly+6, legendX+22, ly+6, item[0])
		fmt.Fprintf(f, `<text x="%d" y="%d" fill="#c8c8e0" font-size="11">%s</text>`,
			legendX+28, ly+10, item[1])
	}

	fmt.Fprintf(f, `</svg>`)
	return nil
}
