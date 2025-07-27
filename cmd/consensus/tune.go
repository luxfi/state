// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/luxfi/genesis/consensus"
)

// runTuning tunes parameters based on network requirements
func runTuning() error {
	scanner := bufio.NewScanner(os.Stdin)
	
	fmt.Println("🎛️  Lux Consensus Parameter Tuning")
	fmt.Println("==================================")
	fmt.Println()
	fmt.Println("Specify any combination of requirements and constraints.")
	fmt.Println("The tool will calculate optimal values for the rest.\n")

	// Start with preset or custom
	params := getStartingParameters(scanner)
	
	// Show current parameters
	fmt.Println("\n📊 Current Parameters:")
	fmt.Println("─────────────────────")
	displayCurrentParams(params)

	// Tuning loop
	for {
		fmt.Println("\n🎯 What would you like to tune?")
		fmt.Println("1. Target finality time (seconds)")
		fmt.Println("2. Number of validators")
		fmt.Println("3. Byzantine fault tolerance (%)")
		fmt.Println("4. Minimum safety cutoff (%)")
		fmt.Println("5. Direct parameter (K, Alpha, Beta)")
		fmt.Println("6. Throughput requirements")
		fmt.Println("7. Show analysis")
		fmt.Println("8. Done")
		
		choice := promptInt(scanner, "Choice", 1, 8, 7)
		
		switch choice {
		case 1:
			params = tuneFinalityTime(scanner, params)
		case 2:
			params = tuneValidatorCount(scanner, params)
		case 3:
			params = tuneByzantineTolerance(scanner, params)
		case 4:
			params = tuneSafetyCutoff(scanner, params)
		case 5:
			params = tuneDirectParameter(scanner, params)
		case 6:
			params = tuneThroughput(scanner, params)
		case 7:
			showAnalysis(params)
		case 8:
			return finalizeTuning(scanner, params)
		}
		
		// Show updated parameters after each change
		fmt.Println("\n📊 Updated Parameters:")
		displayCurrentParams(params)
	}
}

func getStartingParameters(scanner *bufio.Scanner) *consensus.Parameters {
	fmt.Println("Start with:")
	fmt.Println("1. Mainnet preset (21 nodes, 500ms finality)")
	fmt.Println("2. Testnet preset (11 nodes, 600ms finality)")
	fmt.Println("3. Local preset (5 nodes, 200ms finality)")
	fmt.Println("4. Custom")
	
	choice := promptInt(scanner, "Choice", 1, 4, 1)
	
	switch choice {
	case 1:
		return &consensus.MainnetParams
	case 2:
		return &consensus.TestnetParams
	case 3:
		return &consensus.LocalParams
	case 4:
		// Start with sensible defaults
		return &consensus.Parameters{
			K:                     11,
			AlphaPreference:       8,
			AlphaConfidence:       9,
			Beta:                  10,
			ConcurrentRepolls:     10,
			OptimalProcessing:     10,
			MaxOutstandingItems:   256,
			MaxItemProcessingTime: 10 * time.Second,
		}
	}
	return &consensus.MainnetParams
}

func displayCurrentParams(p *consensus.Parameters) {
	fmt.Printf("K=%d, αPref=%d, αConf=%d, β=%d, Pipeline=%d\n", 
		p.K, p.AlphaPreference, p.AlphaConfidence, p.Beta, p.ConcurrentRepolls)
	
	// Calculate and show derived values
	finality := p.CalculateExpectedFinality(50) // Assume 50ms network
	_, confTolerance := p.CalculateFaultTolerance()
	
	fmt.Printf("Expected finality: %.2fs @ 50ms network latency\n", finality.Seconds())
	fmt.Printf("Fault tolerance: %d/%d nodes (%.0f%%)\n", 
		confTolerance, p.K, float64(confTolerance)/float64(p.K)*100)
}

func tuneFinalityTime(scanner *bufio.Scanner, p *consensus.Parameters) *consensus.Parameters {
	fmt.Println("\n⏱️  Finality Time Tuning")
	fmt.Println("────────────────────────")
	
	currentFinality := p.CalculateExpectedFinality(50)
	fmt.Printf("Current finality: %.3fs\n", currentFinality.Seconds())
	
	targetSec := promptFloat(scanner, "Target finality (seconds)", 0.1, 10.0, currentFinality.Seconds())
	targetFinality := time.Duration(targetSec * float64(time.Second))
	
	networkLatMs := promptInt(scanner, "Network latency (ms)", 1, 1000, 50)
	
	// Calculate required Beta
	roundTime := time.Duration(networkLatMs) * time.Millisecond
	requiredBeta := int(targetFinality / roundTime)
	
	if requiredBeta < 4 {
		fmt.Println("⚠️  Minimum Beta is 4 for security. Adjusting...")
		requiredBeta = 4
	}
	
	// Optimize pipelining
	p.Beta = requiredBeta
	p.ConcurrentRepolls = requiredBeta
	
	fmt.Printf("✅ Set Beta=%d with full pipelining\n", p.Beta)
	
	// Offer to adjust K if finality is still not met
	newFinality := p.CalculateExpectedFinality(networkLatMs)
	if math.Abs(newFinality.Seconds()-targetSec) > 0.1 {
		fmt.Printf("⚠️  Achieved finality: %.3fs (target was %.3fs)\n", 
			newFinality.Seconds(), targetSec)
		
		if promptBool(scanner, "Adjust sample size (K) to get closer?", true) {
			// Smaller K can sometimes help with latency
			if p.K > 10 && targetSec < currentFinality.Seconds() {
				p.K = promptInt(scanner, "New K value", 5, p.K-1, p.K-2)
				// Recalculate quorums
				p.AlphaPreference = (p.K * 2 / 3) + 1
				p.AlphaConfidence = (p.K * 3 / 4) + 1
			}
		}
	}
	
	return p
}

func tuneValidatorCount(scanner *bufio.Scanner, p *consensus.Parameters) *consensus.Parameters {
	fmt.Println("\n👥 Validator Count Tuning")
	fmt.Println("─────────────────────────")
	
	totalNodes := promptInt(scanner, "Total number of validators", 3, 1000, 21)
	
	// Adjust K based on network size
	if totalNodes <= 30 {
		p.K = totalNodes
		fmt.Printf("✅ Set K=%d (sampling all nodes)\n", p.K)
	} else {
		// For larger networks, offer choices
		fmt.Println("\nSampling strategy:")
		fmt.Printf("1. Conservative (K=%d) - Higher security\n", min(totalNodes, 50))
		fmt.Printf("2. Balanced (K=%d) - Good trade-off\n", min(totalNodes/2, 30))
		fmt.Printf("3. Performance (K=%d) - Lower overhead\n", min(totalNodes/3, 20))
		fmt.Println("4. Custom")
		
		strategy := promptInt(scanner, "Choice", 1, 4, 2)
		
		switch strategy {
		case 1:
			p.K = min(totalNodes, 50)
		case 2:
			p.K = min(totalNodes/2, 30)
		case 3:
			p.K = min(totalNodes/3, 20)
		case 4:
			p.K = promptInt(scanner, "Sample size (K)", 3, totalNodes, p.K)
		}
		
		fmt.Printf("✅ Set K=%d\n", p.K)
	}
	
	// Recalculate quorums
	p.AlphaPreference = (p.K * 2 / 3) + 1
	p.AlphaConfidence = (p.K * 3 / 4) + 1
	
	// Suggest Beta adjustment for network size
	if totalNodes > 50 && p.Beta < 15 {
		if promptBool(scanner, "Large network detected. Increase Beta for security?", true) {
			p.Beta = promptInt(scanner, "New Beta", p.Beta, 50, 15)
			p.ConcurrentRepolls = min(p.Beta, 20)
		}
	}
	
	return p
}

func tuneByzantineTolerance(scanner *bufio.Scanner, p *consensus.Parameters) *consensus.Parameters {
	fmt.Println("\n🛡️  Byzantine Fault Tolerance Tuning")
	fmt.Println("────────────────────────────────────")
	
	// Show current tolerance
	_, confTolerance := p.CalculateFaultTolerance()
	currentPercent := float64(confTolerance) / float64(p.K) * 100
	
	fmt.Printf("Current tolerance: %d/%d nodes (%.1f%%)\n", 
		confTolerance, p.K, currentPercent)
	
	targetPercent := promptFloat(scanner, "Target Byzantine tolerance (%)", 10, 40, 25)
	
	// Calculate required AlphaConfidence
	maxByzantine := int(float64(p.K) * targetPercent / 100)
	requiredAlpha := p.K - maxByzantine
	
	if requiredAlpha <= p.K/2 {
		fmt.Println("⚠️  Cannot achieve this tolerance - would violate majority requirement")
		requiredAlpha = p.K/2 + 1
	}
	
	p.AlphaConfidence = requiredAlpha
	
	// Ensure AlphaPreference is valid
	if p.AlphaPreference > p.AlphaConfidence {
		p.AlphaPreference = p.AlphaConfidence - 1
		if p.AlphaPreference <= p.K/2 {
			p.AlphaPreference = p.K/2 + 1
		}
	}
	
	fmt.Printf("✅ Set AlphaConfidence=%d (tolerates %d Byzantine nodes)\n", 
		p.AlphaConfidence, p.K-p.AlphaConfidence)
	
	// Calculate new safety cutoff
	safetyCutoff := consensus.RunChecker(p, p.K, 50).SafetyCutoff
	fmt.Printf("📊 New safety cutoff: %.1f%% adversarial stake\n", safetyCutoff)
	
	if safetyCutoff < 25 {
		fmt.Println("⚠️  Low safety cutoff detected!")
		if promptBool(scanner, "Increase Beta for better security?", true) {
			p.Beta = promptInt(scanner, "New Beta", p.Beta+1, 50, p.Beta*2)
			p.ConcurrentRepolls = min(p.Beta, 20)
		}
	}
	
	return p
}

func tuneSafetyCutoff(scanner *bufio.Scanner, p *consensus.Parameters) *consensus.Parameters {
	fmt.Println("\n🎯 Safety Cutoff Tuning")
	fmt.Println("───────────────────────")
	
	// Show current cutoff
	currentCutoff := consensus.RunChecker(p, p.K, 50).SafetyCutoff
	fmt.Printf("Current safety cutoff: %.1f%% (for ε ≤ 10⁻⁹)\n", currentCutoff)
	
	targetCutoff := promptFloat(scanner, "Target safety cutoff (%)", 20, 80, 33)
	
	// This is complex - we need to adjust multiple parameters
	fmt.Println("\nStrategies to increase safety cutoff:")
	fmt.Println("1. Increase Beta (more rounds)")
	fmt.Println("2. Increase AlphaConfidence (higher quorum)")
	fmt.Println("3. Both")
	
	strategy := promptInt(scanner, "Choice", 1, 3, 3)
	
	// Iteratively adjust until we meet target
	iterations := 0
	for consensus.RunChecker(p, p.K, 50).SafetyCutoff < targetCutoff && iterations < 10 {
		iterations++
		
		switch strategy {
		case 1:
			p.Beta += 2
		case 2:
			if p.AlphaConfidence < p.K-1 {
				p.AlphaConfidence++
			}
		case 3:
			p.Beta++
			if p.AlphaConfidence < p.K-1 && iterations%2 == 0 {
				p.AlphaConfidence++
			}
		}
		
		// Update pipelining
		p.ConcurrentRepolls = min(p.Beta, 20)
	}
	
	newCutoff := consensus.RunChecker(p, p.K, 50).SafetyCutoff
	fmt.Printf("✅ Achieved safety cutoff: %.1f%%\n", newCutoff)
	
	return p
}

func tuneDirectParameter(scanner *bufio.Scanner, p *consensus.Parameters) *consensus.Parameters {
	fmt.Println("\n🔧 Direct Parameter Tuning")
	fmt.Println("──────────────────────────")
	
	fmt.Println("Which parameter to adjust?")
	fmt.Println("1. K (sample size)")
	fmt.Println("2. AlphaPreference")
	fmt.Println("3. AlphaConfidence")
	fmt.Println("4. Beta")
	fmt.Println("5. ConcurrentRepolls")
	
	choice := promptInt(scanner, "Choice", 1, 5, 1)
	
	switch choice {
	case 1:
		p.K = promptInt(scanner, "K", 3, 100, p.K)
		// Recalculate quorums if needed
		if p.AlphaPreference > p.K {
			p.AlphaPreference = (p.K * 2 / 3) + 1
		}
		if p.AlphaConfidence > p.K {
			p.AlphaConfidence = (p.K * 3 / 4) + 1
		}
		
	case 2:
		minAlpha := p.K/2 + 1
		p.AlphaPreference = promptInt(scanner, "AlphaPreference", minAlpha, p.K, p.AlphaPreference)
		
	case 3:
		minAlpha := p.AlphaPreference
		p.AlphaConfidence = promptInt(scanner, "AlphaConfidence", minAlpha, p.K, p.AlphaConfidence)
		
	case 4:
		p.Beta = promptInt(scanner, "Beta", 1, 100, p.Beta)
		if promptBool(scanner, "Update pipelining to match?", true) {
			p.ConcurrentRepolls = min(p.Beta, 20)
		}
		
	case 5:
		p.ConcurrentRepolls = promptInt(scanner, "ConcurrentRepolls", 1, p.Beta, p.ConcurrentRepolls)
	}
	
	return p
}

func tuneThroughput(scanner *bufio.Scanner, p *consensus.Parameters) *consensus.Parameters {
	fmt.Println("\n🚀 Throughput Tuning")
	fmt.Println("────────────────────")
	
	targetTPS := promptInt(scanner, "Target transactions per second", 100, 100000, 1000)
	
	// Calculate current throughput
	report := consensus.RunChecker(p, p.K, 50)
	currentTPS := report.ThroughputAnalysis.MaxTransactionsPerSecond
	
	fmt.Printf("Current max TPS: %d\n", currentTPS)
	
	if targetTPS > currentTPS {
		fmt.Println("\nOptions to increase throughput:")
		fmt.Println("1. Increase OptimalProcessing")
		fmt.Println("2. Increase MaxOutstandingItems")
		fmt.Println("3. Reduce finality time")
		fmt.Println("4. All of the above")
		
		choice := promptInt(scanner, "Choice", 1, 4, 4)
		
		switch choice {
		case 1, 4:
			p.OptimalProcessing = promptInt(scanner, "OptimalProcessing", 10, 100, 32)
			if choice != 4 {
				break
			}
			fallthrough
		case 2:
			p.MaxOutstandingItems = promptInt(scanner, "MaxOutstandingItems", 256, 10000, 1024)
			if choice != 4 {
				break
			}
			fallthrough
		case 3:
			// Reduce Beta if possible
			if p.Beta > 4 {
				newBeta := promptInt(scanner, "Reduce Beta to", 4, p.Beta-1, p.Beta/2)
				p.Beta = newBeta
				p.ConcurrentRepolls = newBeta
			}
		}
	}
	
	return p
}

func showAnalysis(p *consensus.Parameters) {
	fmt.Println("\n" + strings.Repeat("─", 60))
	report := consensus.RunChecker(p, p.K, 50)
	fmt.Println(consensus.FormatCheckerReport(report, p.K))
}

func finalizeTuning(scanner *bufio.Scanner, p *consensus.Parameters) error {
	fmt.Println("\n✨ Final Parameters:")
	fmt.Println("───────────────────")
	data, _ := p.ToJSON()
	fmt.Println(string(data))
	
	// Show final analysis
	showAnalysis(p)
	
	// Save if desired
	if promptBool(scanner, "\nSave these parameters?", true) {
		filename := promptString(scanner, "Output filename", "tuned-consensus.json")
		data, _ := p.ToJSON()
		if err := os.WriteFile(filename, data, 0644); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
		fmt.Printf("✅ Parameters saved to %s\n", filename)
		
		// Generate detailed report
		reportFile := strings.TrimSuffix(filename, ".json") + "-report.txt"
		report := consensus.RunChecker(p, p.K, 50)
		reportContent := consensus.FormatCheckerReport(report, p.K)
		if err := os.WriteFile(reportFile, []byte(reportContent), 0644); err != nil {
			fmt.Printf("⚠️  Failed to save report: %v\n", err)
		} else {
			fmt.Printf("📄 Detailed report saved to %s\n", reportFile)
		}
	}
	
	return nil
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}