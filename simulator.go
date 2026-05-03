package main

import (
	"fmt"
	"math/rand/v2"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// SimResult holds the results of a single simulation run.
type SimResult struct {
	FinalBalance        float64
	ReturnPct           float64
	MaxDrawdown         float64
	WinningTrades       int
	WinRate             float64
	EquityCurve         []float64
	MaxWinStreak        int
	MaxLossStreak       int
	MaxTradesToRecovery int
}

// MCResults holds the aggregated results of all Monte Carlo simulations.
type MCResults struct {
	FinalBalances       []float64
	ReturnsPercent      []float64
	MaxDrawdowns        []float64
	WinningTrades       []int
	WinRates            []float64
	EquityCurves        [][]float64
	MaxWinStreaks       []int
	MaxLossStreaks      []int
	MaxTradesToRecovery []int
	ElapsedTime         float64
	SimsPerSecond       float64
}

// Simulator runs Monte Carlo simulations with the given parameters.
type Simulator struct {
	params TradingParameters
	rng    *rand.Rand
}

func NewSimulator(params TradingParameters) *Simulator {
	return &Simulator{
		params: params,
		rng:    rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0)),
	}
}

func (s *Simulator) RunSingle() SimResult {
	p := s.params
	balance := p.InitialBalance
	actualTrades := 0
	var equity []float64
	if p.SaveSVGFile {
		equity = make([]float64, 0, p.TradeCount+1)
		equity = append(equity, balance)
	}
	peakBalance := balance
	maxDrawdown := 0.0
	winStreak, lossStreak := 0, 0
	maxWinStreak, maxLossStreak := 0, 0
	currentTTR, maxTTR := 0, 0
	inDrawdown := false
	winningCount := 0

	for i := 0; i < p.TradeCount; i++ {
		actualTrades++
		// Balance reached zero — simulation is dead
		if balance <= 0 {
			balance = 0
			if p.SaveSVGFile {
				equity = append(equity, balance)
			}
			break
		}
		positionSize := p.InitialBalance * p.RiskPercent
		if p.UseCompounding {
			positionSize = balance * p.RiskPercent
		}
		// Do not risk more than the current balance
		if positionSize > balance {
			positionSize = balance
		}
		roll := s.rng.Float64()
		isWin := roll < p.WinRate
		isBreakeven := !isWin && roll < p.WinRate+p.BreakevenPercent
		if isWin {
			winningCount++
			var rr float64
			switch p.RRModel {
			case "uniform":
				rr = p.WinMultiplier * (1 - p.RRDeviation + s.rng.Float64()*p.RRDeviation*2)
			case "normal":
				rr = p.WinMultiplier * (1 + p.RRSigma*s.rng.NormFloat64())
				if rr < 0 {
					rr = 0
				}
			default: // "fixed"
				rr = p.WinMultiplier
			}
			gross := positionSize * rr
			net := gross - positionSize*p.Commission
			balance += net
			winStreak++
			lossStreak = 0
			if winStreak > maxWinStreak {
				maxWinStreak = winStreak
			}
		} else if isBreakeven {
			// Balance unchanged, only commission is charged
			balance -= positionSize * p.Commission
			if balance < 0 {
				balance = 0
			}
			winStreak = 0
			lossStreak = 0
		} else {
			gross := positionSize
			net := gross + positionSize*p.Commission
			balance -= net
			// Floor at zero — balance cannot go negative
			if balance < 0 {
				balance = 0
			}
			lossStreak++
			winStreak = 0
			if lossStreak > maxLossStreak {
				maxLossStreak = lossStreak
			}
		}
		if p.SaveSVGFile {
			equity = append(equity, balance)
		}
		if balance > peakBalance {
			peakBalance = balance
			if inDrawdown {
				if currentTTR > maxTTR {
					maxTTR = currentTTR
				}
				currentTTR = 0
				inDrawdown = false
			}
		} else {
			if !inDrawdown {
				inDrawdown = true
				currentTTR = 0
			}
			currentTTR++
		}
		if peakBalance > 0 {
			dd := (peakBalance - balance) / peakBalance
			if dd > maxDrawdown {
				maxDrawdown = dd
			}
		}
	}
	if inDrawdown && currentTTR > maxTTR {
		maxTTR = currentTTR
	}
	actualWinRate := 0.0
	if actualTrades > 0 {
		actualWinRate = float64(winningCount) / float64(actualTrades)
	}
	return SimResult{
		FinalBalance:        balance,
		ReturnPct:           (balance - p.InitialBalance) / p.InitialBalance,
		MaxDrawdown:         maxDrawdown,
		WinningTrades:       winningCount,
		WinRate:             actualWinRate,
		EquityCurve:         equity,
		MaxWinStreak:        maxWinStreak,
		MaxLossStreak:       maxLossStreak,
		MaxTradesToRecovery: maxTTR,
	}
}

func (s *Simulator) RunMonteCarlo() MCResults {
	n := s.params.SimulationCount
	numCPU := runtime.NumCPU()
	fmt.Printf("Running %d simulations on %d cores...\n", n, numCPU)
	start := time.Now()

	res := MCResults{
		FinalBalances:       make([]float64, n),
		ReturnsPercent:      make([]float64, n),
		MaxDrawdowns:        make([]float64, n),
		WinRates:            make([]float64, n),
		MaxWinStreaks:       make([]int, n),
		MaxLossStreaks:      make([]int, n),
		MaxTradesToRecovery: make([]int, n),
		WinningTrades:       make([]int, n),
	}

	// Allocate equity curves only if SVG output is enabled
	if s.params.SaveSVGFile {
		res.EquityCurves = make([][]float64, n)
	}

	var completed atomic.Int64
	var wg sync.WaitGroup

	jobs := make(chan int, n)
	for i := 0; i < n; i++ {
		jobs <- i
	}
	close(jobs)

	for w := 0; w < numCPU; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			rng := rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), uint64(workerID)))
			workerSim := &Simulator{
				params: s.params,
				rng:    rng,
			}

			for i := range jobs {
				r := workerSim.RunSingle()
				res.FinalBalances[i] = r.FinalBalance
				res.ReturnsPercent[i] = r.ReturnPct
				res.MaxDrawdowns[i] = r.MaxDrawdown
				res.WinRates[i] = r.WinRate
				res.MaxWinStreaks[i] = r.MaxWinStreak
				res.MaxLossStreaks[i] = r.MaxLossStreak
				res.MaxTradesToRecovery[i] = r.MaxTradesToRecovery
				res.WinningTrades[i] = r.WinningTrades
				// Store equity curve only if SVG output is enabled
				if workerSim.params.SaveSVGFile {
					res.EquityCurves[i] = r.EquityCurve
				}
				done := completed.Add(1)
				if done%100 == 0 {
					fmt.Printf("  Completed: %d/%d\n", done, n)
				}
			}
		}(w)
	}

	wg.Wait()

	elapsed := time.Since(start).Seconds()
	res.ElapsedTime = elapsed
	res.SimsPerSecond = float64(n) / elapsed

	fmt.Printf("\n[OK] Completed in %.2f sec\n", elapsed)
	fmt.Printf("[OK] Speed: %.0f simulations/sec\n\n", res.SimsPerSecond)

	return res
}
