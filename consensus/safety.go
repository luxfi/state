// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package consensus

import (
	"fmt"
	"strings"
)

// SafetyLevel represents the safety assessment of consensus parameters
type SafetyLevel int

const (
	SafetyOptimal SafetyLevel = iota
	SafetyGood
	SafetyWarning
	SafetyCritical
	SafetyDanger
)

func (s SafetyLevel) String() string {
	switch s {
	case SafetyOptimal:
		return "OPTIMAL"
	case SafetyGood:
		return "GOOD"
	case SafetyWarning:
		return "WARNING"
	case SafetyCritical:
		return "CRITICAL"
	case SafetyDanger:
		return "DANGER"
	default:
		return "UNKNOWN"
	}
}

// SafetyReport contains the safety analysis of consensus parameters
type SafetyReport struct {
	Level        SafetyLevel
	Issues       []string
	Warnings     []string
	Suggestions  []string
	Explanation  string
}

// AnalyzeSafety performs a comprehensive safety analysis of consensus parameters
func AnalyzeSafety(p *Parameters, totalNodes int) SafetyReport {
	report := SafetyReport{
		Level:       SafetyOptimal,
		Issues:      []string{},
		Warnings:    []string{},
		Suggestions: []string{},
	}

	// Check K vs total nodes
	if p.K > totalNodes {
		report.Level = SafetyDanger
		report.Issues = append(report.Issues, 
			fmt.Sprintf("K (%d) cannot exceed total nodes (%d)", p.K, totalNodes))
	} else if p.K < totalNodes/2 {
		report.Level = SafetyWarning
		report.Warnings = append(report.Warnings,
			fmt.Sprintf("K (%d) is less than 50%% of nodes (%d), reducing safety", p.K, totalNodes))
	}

	// Check quorum thresholds
	minSafeAlpha := (p.K / 2) + 1
	if p.AlphaPreference < minSafeAlpha {
		report.Level = SafetyDanger
		report.Issues = append(report.Issues,
			fmt.Sprintf("AlphaPreference (%d) must be > K/2 (%d) for safety", 
				p.AlphaPreference, p.K/2))
	}

	if p.AlphaConfidence < p.AlphaPreference {
		report.Level = SafetyCritical
		report.Issues = append(report.Issues,
			"AlphaConfidence must be >= AlphaPreference")
	}

	// Analyze fault tolerance
	prefTolerance, confTolerance := p.CalculateFaultTolerance()
	
	// Check if fault tolerance is reasonable
	if float64(confTolerance)/float64(totalNodes) < 0.1 && totalNodes > 5 {
		report.Level = max(report.Level, SafetyWarning)
		report.Warnings = append(report.Warnings,
			fmt.Sprintf("Low fault tolerance: can only tolerate %d failures out of %d nodes (%.1f%%)",
				confTolerance, totalNodes, float64(confTolerance)/float64(totalNodes)*100))
	}

	// Check Beta (finalization rounds)
	if p.Beta < 4 {
		report.Level = max(report.Level, SafetyWarning)
		report.Warnings = append(report.Warnings,
			"Beta < 4 may compromise finality guarantees")
		report.Suggestions = append(report.Suggestions,
			"Consider increasing Beta to at least 4 for production use")
	} else if p.Beta > 100 {
		report.Level = max(report.Level, SafetyWarning)
		report.Warnings = append(report.Warnings,
			fmt.Sprintf("Beta = %d is very high, may cause slow finality", p.Beta))
	}

	// Check pipelining
	if p.ConcurrentRepolls > p.Beta {
		report.Warnings = append(report.Warnings,
			"ConcurrentRepolls > Beta has no benefit")
		report.Suggestions = append(report.Suggestions,
			fmt.Sprintf("Set ConcurrentRepolls to %d (same as Beta)", p.Beta))
	} else if p.ConcurrentRepolls < p.Beta/4 && p.Beta > 8 {
		report.Suggestions = append(report.Suggestions,
			"Consider increasing ConcurrentRepolls for better throughput")
	}

	// Check processing limits
	if p.MaxOutstandingItems < 10 {
		report.Level = max(report.Level, SafetyWarning)
		report.Warnings = append(report.Warnings,
			"MaxOutstandingItems < 10 may severely limit throughput")
	}

	// Generate explanation
	report.Explanation = generateSafetyExplanation(p, totalNodes, prefTolerance, confTolerance)

	return report
}

func generateSafetyExplanation(p *Parameters, totalNodes, prefTolerance, confTolerance int) string {
	var parts []string

	// Sampling explanation
	samplePercent := float64(p.K) / float64(totalNodes) * 100
	parts = append(parts, fmt.Sprintf(
		"Sampling %d out of %d nodes (%.1f%%) per round",
		p.K, totalNodes, samplePercent))

	// Quorum explanation
	prefPercent := float64(p.AlphaPreference) / float64(p.K) * 100
	confPercent := float64(p.AlphaConfidence) / float64(p.K) * 100
	parts = append(parts, fmt.Sprintf(
		"Preference changes with %d/%d votes (%.1f%%), finalization requires %d/%d votes (%.1f%%)",
		p.AlphaPreference, p.K, prefPercent,
		p.AlphaConfidence, p.K, confPercent))

	// Fault tolerance
	parts = append(parts, fmt.Sprintf(
		"Can tolerate %d failures for liveness, %d for safety",
		prefTolerance, confTolerance))

	// Finality time estimate
	finality50ms := p.CalculateExpectedFinality(50)
	finality10ms := p.CalculateExpectedFinality(10)
	parts = append(parts, fmt.Sprintf(
		"Expected finality: %.2fs (50ms latency), %.2fs (10ms latency)",
		finality50ms.Seconds(), finality10ms.Seconds()))

	// Security assessment
	if confPercent >= 80 {
		parts = append(parts, "Strong safety guarantee with supermajority requirement")
	} else if confPercent >= 67 {
		parts = append(parts, "Good safety with classical BFT-level security")
	} else {
		parts = append(parts, "Moderate safety - consider increasing AlphaConfidence")
	}

	return strings.Join(parts, ". ")
}

// RecommendParameters suggests optimal parameters based on network characteristics
func RecommendParameters(totalNodes int, targetFinality float64, networkLatencyMs int) (*Parameters, SafetyReport) {
	builder := NewBuilder()
	
	// Start with node count optimization
	builder = builder.ForNodeCount(totalNodes)
	
	// Apply target finality if specified
	if targetFinality > 0 {
		builder = builder.WithTargetFinality(
			time.Duration(targetFinality*1000) * time.Millisecond,
			networkLatencyMs)
	}
	
	// Build parameters
	params, _ := builder.Build()
	
	// Generate safety report
	report := AnalyzeSafety(params, totalNodes)
	
	// Add recommendations based on node count
	if totalNodes <= 5 {
		report.Suggestions = append(report.Suggestions,
			"For small networks (<=5 nodes), consider using all nodes (K="+fmt.Sprintf("%d", totalNodes)+")")
	} else if totalNodes > 100 {
		report.Suggestions = append(report.Suggestions,
			"For large networks (>100 nodes), consider capping K at 50-100 for performance")
	}
	
	return params, report
}

// ValidateForProduction checks if parameters are suitable for production use
func ValidateForProduction(p *Parameters, totalNodes int) error {
	report := AnalyzeSafety(p, totalNodes)
	
	if report.Level >= SafetyCritical {
		return fmt.Errorf("parameters not safe for production: %s", 
			strings.Join(report.Issues, "; "))
	}
	
	// Additional production checks
	if p.Beta < 4 {
		return fmt.Errorf("Beta must be at least 4 for production use")
	}
	
	if float64(p.AlphaConfidence)/float64(p.K) < 0.67 {
		return fmt.Errorf("AlphaConfidence must be at least 67%% of K for production")
	}
	
	confTolerance := p.K - p.AlphaConfidence
	if float64(confTolerance)/float64(totalNodes) > 0.33 {
		return fmt.Errorf("production networks should not tolerate more than 33%% Byzantine nodes")
	}
	
	return nil
}

func max(a, b SafetyLevel) SafetyLevel {
	if a > b {
		return a
	}
	return b
}