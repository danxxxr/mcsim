package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// InteractiveSetup prompts the user to enter simulation parameters.
// Press Enter to keep the current value from config.
func InteractiveSetup(p TradingParameters) (TradingParameters, bool) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println("# INTERACTIVE SETUP")
	fmt.Println("# Press Enter to keep current value")
	fmt.Println(strings.Repeat("=", 70))
	p.InitialBalance = readFloat(reader, "Initial balance", p.InitialBalance, "%.0f")
	p.WinRate = readFloat(reader, "Win rate (0.65 = 65%)", p.WinRate, "%.2f")
	p.BreakevenPercent = readFloat(reader, "Breakeven (0.05 = 5%, 0 = disabled)", p.BreakevenPercent, "%.2f")
	p.WinMultiplier = readFloat(reader, "Reward:risk (1.5 = 1.5:1)", p.WinMultiplier, "%.2f")
	p.RiskPercent = readFloat(reader, "Risk per trade (0.01 = 1%)", p.RiskPercent, "%.2f")
	p.TradeCount = readInt(reader, "Trade count", p.TradeCount)
	p.SimulationCount = readInt(reader, "Simulation count", p.SimulationCount)
	p.Commission = readFloat(reader, "Commission (0.01 = 1%, 0 = disabled)", p.Commission, "%.2f")
	p.RuinThreshold = readFloat(reader, "Ruin threshold in $ (0 = disabled)", p.RuinThreshold, "%.0f")

	// Use compounding — prompt as y/n
	compounding := "n"
	if p.UseCompounding {
		compounding = "y"
	}
	fmt.Printf("Use compounding [%s]: ", compounding)
	input := strings.TrimSpace(readLine(reader))
	if input != "" {
		p.UseCompounding = input == "y" || input == "yes" || input == "1"
	}

	p.RRModel = readString(reader, "RR model (fixed/uniform/normal)", p.RRModel)

	// Show deviation/sigma only for the selected RR model
	switch p.RRModel {
	case "uniform":
		p.RRDeviation = readFloat(reader, "RR deviation (0.1 = ±10%%)", p.RRDeviation, "%.2f")
	case "normal":
		p.RRSigma = readFloat(reader, "RR sigma (0.1 = 10%%)", p.RRSigma, "%.2f")
	}

	// Validate params and show errors/warnings
	errors, warnings := ValidateParams(p)

	if len(warnings) > 0 {
		fmt.Println()
		for _, w := range warnings {
			fmt.Printf("[!] %s\n", w)
		}
	}

	if len(errors) > 0 {
		fmt.Println()
		for _, e := range errors {
			fmt.Printf("[X] %s\n", e)
		}
		fmt.Println("[->] Fix the values and try again")
		return InteractiveSetup(p)
	}

	// Ask if user wants to run stress test
	fmt.Printf("Run stress test? [n]: ")
	input = strings.TrimSpace(readLine(reader))
	runStress := input == "y" || input == "yes" || input == "1"

	return p, runStress
}

func readFloat(reader *bufio.Reader, prompt string, current float64, format string) float64 {
	fmt.Printf("%s ["+format+"]: ", prompt, current)
	input := strings.TrimSpace(readLine(reader))
	if input == "" {
		return current
	}
	if v, err := strconv.ParseFloat(input, 64); err == nil {
		return v
	}
	fmt.Println("[!] Invalid value, keeping current")
	return current
}

func readInt(reader *bufio.Reader, prompt string, current int) int {
	fmt.Printf("%s [%d]: ", prompt, current)
	input := strings.TrimSpace(readLine(reader))
	if input == "" {
		return current
	}
	if v, err := strconv.Atoi(input); err == nil {
		return v
	}
	fmt.Println("[!] Invalid value, keeping current")
	return current
}

func readLine(reader *bufio.Reader) string {
	line, _ := reader.ReadString('\n')
	return strings.TrimRight(line, "\r\n")
}

func readString(reader *bufio.Reader, prompt string, current string) string {
	fmt.Printf("%s [%s]: ", prompt, current)
	input := strings.TrimSpace(readLine(reader))
	if input == "" {
		return current
	}
	return input
}
