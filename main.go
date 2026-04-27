package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {

	// CLI flags
	configPath := flag.String("config", "config.ini", "Path to the configuration file")
	showVersion := flag.Bool("version", false, "Show version")
	showHelp := flag.Bool("help", false, "Show help")
	fBalance := flag.Float64("balance", 0, "Initial balance")
	fWinRate := flag.Float64("win-rate", 0, "Win rate % (0.65 = 65%)")
	flag.Float64Var(fWinRate, "w", 0, "Win rate % (alias -win-rate)")
	fMultiplier := flag.Float64("rr", 0, "Reward:risk (1.5 = 1.5:1)")
	fRisk := flag.Float64("risk", 0, "Risk per trade % (0.01 = 1%)")
	flag.Float64Var(fRisk, "r", 0, "Risk per trade % (alias -risk)")
	fTrades := flag.Int("trades", 0, "Trade count")
	flag.IntVar(fTrades, "t", 0, "Trade count (alias -trades)")
	fSims := flag.Int("sims", 0, "Simulation count")
	flag.IntVar(fSims, "s", 0, "Simulation count (alias -sims)")
	fCommission := flag.Float64("commission", 0, "Broker commission (0.01 = 1%)")
	fCompound := flag.Bool("compounding", false, "Use compounding")
	fRRModel := flag.String("rr-model", "", "RR model: fixed, uniform, normal")
	fRRDeviation := flag.Float64("rr-deviation", 0, "Deviation for uniform (0.1 = ±10%)")
	fRRSigma := flag.Float64("rr-sigma", 0, "Standard deviation for normal (0.1 = 10%)")
	fSaveReport := flag.Bool("save-report", false, "Save a text report")
	flag.BoolVar(fSaveReport, "sr", false, "Save a text report (alias -save-report)")
	fSaveCSV := flag.Bool("save-csv", false, "Save CSV file")
	flag.BoolVar(fSaveCSV, "sc", false, "Save CSV file (alias -save-csv)")
	fSaveSVG := flag.Bool("save-svg", false, "Save SVG chart")
	flag.BoolVar(fSaveSVG, "ss", false, "Save SVG chart (alias -save-svg)")
	fSVGMaxCurves := flag.Int("svg-max-curves", 0, "Maximum curves on the SVG chart")
	fSaveNone := flag.Bool("no-save", false, "Do not save anything, output to console only")
	flag.BoolVar(fSaveNone, "n", false, "Do not save anything, output to console only (alias -no-save)")
	fSaveAll := flag.Bool("save-all", false, "Save all (report, CSV, SVG)")
	flag.BoolVar(fSaveAll, "sa", false, "Save all (report, CSV, SVG) (alias -save-all)")
	fOutputDir := flag.String("output", ".", "Directory for saving files")
	flag.StringVar(fOutputDir, "o", ".", "Directory for saving files (alias -output)")

	flag.Parse()

	if *showVersion {
		fmt.Println("mcsim version 1.00")
		os.Exit(0)
	}

	if *showHelp {
		flag.Usage()
		os.Exit(0)
	}

	if _, err := os.Stat(*configPath); os.IsNotExist(err) {
		if err := WriteDefaultConfig(*configPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating config.ini: %v\n", err)
		} else {
			fmt.Printf("[OK] Configuration file created: %s\n", *configPath)
		}
	}

	params, parseErrors, err := LoadConfig(*configPath)
	if err != nil {
		fmt.Printf("[!] Failed to read %s: %v — using default values\n", *configPath, err)
		params = DefaultParams()
	} else {
		fmt.Printf("[OK] Parameters loaded from %s\n\n", *configPath)
	}

	// no-save and save-all take priority
	if *fSaveNone {
		params.SaveReport = false
		params.SaveCSVFile = false
		params.SaveSVGFile = false
	}
	if *fSaveAll {
		params.SaveReport = true
		params.SaveCSVFile = true
		params.SaveSVGFile = true
	}

	// Individual flags override config only if explicitly set
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "balance":
			params.InitialBalance = *fBalance
		case "win-rate", "w":
			params.WinRate = *fWinRate
		case "rr":
			params.WinMultiplier = *fMultiplier
		case "risk", "r":
			params.RiskPercent = *fRisk
		case "trades", "t":
			params.TradeCount = *fTrades
		case "sims", "s":
			params.SimulationCount = *fSims
		case "commission":
			params.Commission = *fCommission
		case "compounding":
			params.UseCompounding = *fCompound
		case "rr-model":
			params.RRModel = *fRRModel
		case "rr-deviation":
			params.RRDeviation = *fRRDeviation
		case "rr-sigma":
			params.RRSigma = *fRRSigma
		case "save-report", "sr":
			params.SaveReport = *fSaveReport
		case "save-csv", "sc":
			params.SaveCSVFile = *fSaveCSV
		case "save-svg", "ss":
			params.SaveSVGFile = *fSaveSVG
		case "svg-max-curves":
			params.SVGMaxCurves = *fSVGMaxCurves
		case "output", "o":
			params.OutputDir = *fOutputDir
		}
	})

	// Stop if there are parse errors
	if len(parseErrors) > 0 {
		fmt.Println(strings.Repeat("=", 70))
		fmt.Println("[X] CONFIGURATION FILE ERRORS — simulation not started")
		fmt.Println(strings.Repeat("=", 70))
		for _, e := range parseErrors {
			fmt.Println(e)
		}
		fmt.Println()
		fmt.Printf("[->] Fix %s and run again\n", *configPath)
		os.Exit(1)
	}
	validationErrors, validationWarnings := ValidateParams(params)

	// Print warnings first
	if len(validationWarnings) > 0 {
		fmt.Println(strings.Repeat("=", 70))
		fmt.Println("[!] WARNINGS")
		fmt.Println(strings.Repeat("=", 70))
		for _, w := range validationWarnings {
			fmt.Println(w)
		}
		fmt.Println()
	}

	// Stop if there are critical errors
	if len(validationErrors) > 0 {
		fmt.Println(strings.Repeat("=", 70))
		fmt.Println("[X] CONFIGURATION ERRORS — simulation not started")
		fmt.Println(strings.Repeat("=", 70))
		for _, e := range validationErrors {
			fmt.Println(e)
		}
		fmt.Println()
		fmt.Printf("[->] Fix %s and run again\n", *configPath)
		os.Exit(1)
	}

	fmt.Println(strings.Repeat("=", 70))
	fmt.Println("# MONTE CARLO TRADING SYSTEM SIMULATOR")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Printf("Parameters:\n")
	fmt.Printf("• Initial balance:    $%.0f\n", params.InitialBalance)
	fmt.Printf("• Win rate:           %.1f%%\n", params.WinRate*100)
	fmt.Printf("• Reward:risk:        %.2f\n", params.WinMultiplier)
	fmt.Printf("• Risk per trade:     %.2f%%\n", params.RiskPercent*100)
	fmt.Printf("• Trade count:        %d\n", params.TradeCount)
	fmt.Printf("• Simulation count:   %d\n\n", params.SimulationCount)

	sim := NewSimulator(params)
	results := sim.RunMonteCarlo()

	report := sim.GenerateReport(results)
	fmt.Println(report)

	// Create output directory before saving
	if params.SaveReport || params.SaveCSVFile || params.SaveSVGFile {
		if err := os.MkdirAll(params.OutputDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating directory %s: %v\n", params.OutputDir, err)
			os.Exit(1)
		}
	}

	if params.SaveReport {
		reportPath := filepath.Join(params.OutputDir, "monte_carlo_report.txt")
		if err := os.WriteFile(reportPath, []byte(report), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Report error: %v\n", err)
		} else {
			fmt.Printf("[OK] Report saved: %s\n", reportPath)
		}
	}

	if params.SaveCSVFile {
		csvPath := filepath.Join(params.OutputDir, "monte_carlo_results.csv")
		if err := SaveCSV(results, csvPath); err != nil {
			fmt.Fprintf(os.Stderr, "CSV error: %v\n", err)
		} else {
			fmt.Printf("[OK] CSV saved: %s\n", csvPath)
		}
	}

	if params.SaveSVGFile {
		svgPath := filepath.Join(params.OutputDir, "monte_carlo_results.svg")
		fmt.Println("\nGenerating chart...")
		if err := SaveSVG(results, params, svgPath); err != nil {
			fmt.Fprintf(os.Stderr, "SVG error: %v\n", err)
		} else {
			fmt.Printf("[OK] Chart saved: %s\n", svgPath)
		}
	}

	fmt.Println("\n[OK] Done!")
}
