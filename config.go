package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type TradingParameters struct {
	InitialBalance   float64
	WinRate          float64
	BreakevenPercent float64
	WinMultiplier    float64
	RiskPercent      float64
	TradeCount       int
	SimulationCount  int
	Commission       float64
	UseCompounding   bool
	SaveReport       bool
	SaveCSVFile      bool
	SaveSVGFile      bool
	RRModel          string
	RRDeviation      float64
	RRSigma          float64
	SVGMaxCurves     int
	OutputDir        string
}

func DefaultParams() TradingParameters {
	return TradingParameters{
		InitialBalance:   10000,
		WinRate:          0.65,
		BreakevenPercent: 0.0,
		WinMultiplier:    1.0,
		RiskPercent:      0.01,
		TradeCount:       100,
		SimulationCount:  1000,
		Commission:       0.0,
		UseCompounding:   true,
		SaveReport:       true,
		SaveCSVFile:      true,
		SaveSVGFile:      true,
		RRModel:          "fixed",
		RRDeviation:      0.1,
		RRSigma:          0.1,
		SVGMaxCurves:     200,
		OutputDir:        ".",
	}
}

// ValidateParams validates the parameters.
// Returns a list of critical errors and warnings.
// If errors is non-empty — simulation cannot be started.
func ValidateParams(p TradingParameters) (errors []string, warnings []string) {
	// Critical errors (simulation is pointless)
	if p.WinRate < 0 || p.WinRate > 1 {
		errors = append(errors,
			fmt.Sprintf("win_rate = %.4f — must be in range [0.0, 1.0]", p.WinRate))
	}

	if p.BreakevenPercent < 0 {
		errors = append(errors,
			fmt.Sprintf("breakeven_percent = %.4f — must be greater than or equal to 0", p.BreakevenPercent))
	}

	if p.BreakevenPercent+p.WinRate > 1 {
		errors = append(errors,
			fmt.Sprintf("win_rate + breakeven_percent = %.4f — cannot exceed 1.0", p.WinRate+p.BreakevenPercent))
	}

	if p.RiskPercent <= 0 {
		errors = append(errors,
			fmt.Sprintf("risk_percent = %.4f — must be greater than 0", p.RiskPercent))
	}

	if p.RiskPercent > 1 {
		errors = append(errors,
			fmt.Sprintf("risk_percent = %.4f — cannot exceed 1.0 (100%% of balance)", p.RiskPercent))
	}

	if p.InitialBalance <= 0 {
		errors = append(errors,
			fmt.Sprintf("initial_balance = %.2f — must be greater than 0", p.InitialBalance))
	}

	if p.WinMultiplier <= 0 {
		errors = append(errors,
			fmt.Sprintf("win_multiplier = %.4f — must be greater than 0", p.WinMultiplier))
	}

	if p.TradeCount <= 0 {
		errors = append(errors,
			fmt.Sprintf("trade_count = %d — must be greater than 0", p.TradeCount))
	}

	if p.SimulationCount <= 0 {
		errors = append(errors,
			fmt.Sprintf("simulation_count = %d — must be greater than 0", p.SimulationCount))
	}

	if p.RRModel != "fixed" && p.RRModel != "uniform" && p.RRModel != "normal" {
		errors = append(errors,
			fmt.Sprintf("rr_model = %q — expected: fixed, uniform, normal", p.RRModel))
	}

	if p.RRSigma <= 0 {
		errors = append(errors,
			fmt.Sprintf("rr_sigma = %.4f — must be greater than 0", p.RRSigma))
	}

	if p.RRDeviation <= 0 {
		errors = append(errors,
			fmt.Sprintf("rr_deviation = %.4f — must be greater than 0", p.RRDeviation))
	}

	if p.SVGMaxCurves <= 0 {
		errors = append(errors,
			fmt.Sprintf("svg_max_curves = %d — must be greater than 0", p.SVGMaxCurves))
	}
	// Warnings (simulation will run, but results may be unexpected)
	if p.Commission < 0 {
		warnings = append(warnings,
			fmt.Sprintf("commission = %.4f is negative — commission will increase profit", p.Commission))
	}

	if p.WinRate > 0.95 {
		warnings = append(warnings,
			fmt.Sprintf("win_rate = %.4f is very high (> 95%%) — results may be unrealistic", p.WinRate))
	}

	if p.RiskPercent > 0.25 {
		warnings = append(warnings,
			fmt.Sprintf("risk_percent = %.4f is very high (> 25%%) — high probability of ruin", p.RiskPercent))
	}

	if p.SimulationCount < 100 {
		warnings = append(warnings,
			fmt.Sprintf("simulation_count = %d is too low (< 100) — statistics will be unreliable", p.SimulationCount))
	}

	if p.RRModel == "uniform" && p.RRDeviation >= 1 {
		warnings = append(warnings,
			fmt.Sprintf("rr_deviation = %.4f is very large (>= 100%%) — RR may become negative", p.RRDeviation))
	}

	if p.RRModel == "normal" && p.RRSigma >= 0.5 {
		warnings = append(warnings,
			fmt.Sprintf("rr_sigma = %.4f is very large (>= 50%%) — extreme RR values are possible", p.RRSigma))
	}

	return errors, warnings
}

// LoadConfig reads parameters from an INI file.
func LoadConfig(path string) (TradingParameters, []string, error) {
	p := DefaultParams()
	var parseErrors []string

	f, err := os.Open(path)
	if err != nil {
		return p, nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		rawLine := strings.TrimSpace(scanner.Text())
		if rawLine == "" || strings.HasPrefix(rawLine, "#") ||
			strings.HasPrefix(rawLine, ";") || strings.HasPrefix(rawLine, "[") {
			continue
		}
		parts := strings.SplitN(rawLine, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if idx := strings.Index(val, "#"); idx >= 0 {
			val = strings.TrimSpace(val[:idx])
		}
		if idx := strings.Index(val, ";"); idx >= 0 {
			val = strings.TrimSpace(val[:idx])
		}

		switch key {
		case "initial_balance":
			if v, err := strconv.ParseFloat(val, 64); err == nil {
				p.InitialBalance = v
			} else {
				parseErrors = append(parseErrors,
					fmt.Sprintf("initial_balance = %q — expected a number (e.g. 10000)", val))
			}

		case "win_rate":
			if v, err := strconv.ParseFloat(val, 64); err == nil {
				p.WinRate = v
			} else {
				parseErrors = append(parseErrors,
					fmt.Sprintf("win_rate = %q — expected a number (e.g. 0.65)", val))
			}

		case "breakeven_percent":
			if v, err := strconv.ParseFloat(val, 64); err == nil {
				p.BreakevenPercent = v
			} else {
				parseErrors = append(parseErrors,
					fmt.Sprintf("breakeven_percent = %q — expected a number (e.g. 0.1)", val))
			}

		case "win_multiplier":
			if v, err := strconv.ParseFloat(val, 64); err == nil {
				p.WinMultiplier = v
			} else {
				parseErrors = append(parseErrors,
					fmt.Sprintf("win_multiplier = %q — expected a number (e.g. 1.5)", val))
			}

		case "risk_percent":
			if v, err := strconv.ParseFloat(val, 64); err == nil {
				p.RiskPercent = v
			} else {
				parseErrors = append(parseErrors,
					fmt.Sprintf("risk_percent = %q — expected a number (e.g. 0.01)", val))
			}

		case "trade_count":
			if v, err := strconv.Atoi(val); err == nil {
				p.TradeCount = v
			} else {
				parseErrors = append(parseErrors,
					fmt.Sprintf("trade_count = %q — expected an integer (e.g. 100)", val))
			}

		case "simulation_count":
			if v, err := strconv.Atoi(val); err == nil {
				p.SimulationCount = v
			} else {
				parseErrors = append(parseErrors,
					fmt.Sprintf("simulation_count = %q — expected an integer (e.g. 1000)", val))
			}

		case "commission":
			if v, err := strconv.ParseFloat(val, 64); err == nil {
				p.Commission = v
			} else {
				parseErrors = append(parseErrors,
					fmt.Sprintf("commission = %q — expected a number (e.g. 0.001)", val))
			}

		case "use_compounding":
			p.UseCompounding = val == "true" || val == "1" || val == "yes"
		case "save_report":
			p.SaveReport = val == "true" || val == "1" || val == "yes"
		case "save_csv":
			p.SaveCSVFile = val == "true" || val == "1" || val == "yes"
		case "save_svg":
			p.SaveSVGFile = val == "true" || val == "1" || val == "yes"

		case "rr_model":
			if val == "fixed" || val == "uniform" || val == "normal" {
				p.RRModel = val
			} else {
				parseErrors = append(parseErrors,
					fmt.Sprintf("rr_model = %q — expected: fixed, uniform, normal", val))
			}

		case "rr_deviation":
			if v, err := strconv.ParseFloat(val, 64); err == nil {
				p.RRDeviation = v
			} else {
				parseErrors = append(parseErrors,
					fmt.Sprintf("rr_deviation = %q — expected a number (e.g. 0.1)", val))
			}

		case "rr_sigma":
			if v, err := strconv.ParseFloat(val, 64); err == nil {
				p.RRSigma = v
			} else {
				parseErrors = append(parseErrors,
					fmt.Sprintf("rr_sigma = %q — expected a number (e.g. 0.1)", val))
			}
		case "svg_max_curves":
			if v, err := strconv.Atoi(val); err == nil {
				p.SVGMaxCurves = v
			} else {
				parseErrors = append(parseErrors,
					fmt.Sprintf("svg_max_curves = %q — expected an integer (e.g. 200)", val))
			}
		case "output_dir":
			p.OutputDir = val
		}
	}

	return p, parseErrors, scanner.Err()
}

// WriteDefaultConfig creates config.ini with detailed comments.
func WriteDefaultConfig(path string) error {
	content := `# ============================================================
# Monte Carlo Simulator — Configuration file
# ============================================================
# Lines starting with # or ; are comments (ignored)
# Format: key = value

[simulation]

# Initial deposit in dollars
initial_balance = 10000

# Win rate (0.65 = 65%)
win_rate = 0.65

# Breakeven trades rate (0.05 = 5%), 0 = disabled
breakeven_percent = 0.0

# Reward:risk ratio (1.5 = 1.5:1)
win_multiplier = 1.0

# Risk per trade (0.01 = 1% of balance)
risk_percent = 0.01

# Number of trades per simulation
trade_count = 100

# Number of Monte Carlo simulations
simulation_count = 1000

# Broker commission (0.01 = 1%), 0 = no commission
commission = 0.0

# Reinvest profits: true = compounding, false = fixed size
use_compounding = true

# RR model for winning trades:
#   fixed   — fixed RR (always exactly win_multiplier)
#   uniform — uniform distribution (rr_deviation = ±deviation, e.g. 0.1 = ±10%)
#   normal  — normal distribution (rr_sigma = standard deviation, e.g. 0.1 = ±10%)
rr_model = fixed

# Deviation for uniform (0.1 = ±10%)
rr_deviation = 0.1

# Standard deviation for normal (0.1 = 10%)
rr_sigma = 0.1

[output]

# Save text report (monte_carlo_report.txt)
save_report = true

# Save CSV with all simulation data (monte_carlo_results.csv)
save_csv = true

# Save SVG chart of equity curves (monte_carlo_results.svg)
# Note: large simulation_count, trade_count and svg_max_curves requires a lot of memory
# simulation_count=100000, trade_count=1000, svg_max_curves=200 → ~800 MB
save_svg = true

# Maximum background curves on the chart (fewer = smaller file size)
svg_max_curves = 60

# Directory for saving files (. = current directory)
output_dir = .
`
	return os.WriteFile(path, []byte(content), 0644)
}
