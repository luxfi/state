// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package consensus

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// ParameterGuide provides guidance on parameter selection
type ParameterGuide struct {
	Parameter   string
	Description string
	Formula     string
	MinValue    interface{}
	MaxValue    interface{}
	Typical     string
	Impact      string
	TradeOffs   string
}

// GetParameterGuides returns comprehensive guidance for all parameters
func GetParameterGuides() []ParameterGuide {
	return []ParameterGuide{
		{
			Parameter:   "K (Sample Size)",
			Description: "Number of validators randomly sampled in each consensus round",
			Formula:     "K ≤ TotalNodes; typically K = min(TotalNodes, 20-50)",
			MinValue:    1,
			MaxValue:    "TotalNodes",
			Typical:     "5-21 for small networks, 20-50 for large networks",
			Impact:      "Higher K = stronger statistical guarantees but more network overhead",
			TradeOffs:   "Security vs Performance: Larger K increases message complexity O(K) per round",
		},
		{
			Parameter:   "AlphaPreference",
			Description: "Quorum threshold for changing preference (liveness threshold)",
			Formula:     "(K/2) < AlphaPreference ≤ K; typically 60-75% of K",
			MinValue:    "K/2 + 1",
			MaxValue:    "K",
			Typical:     "≈67% of K for good liveness",
			Impact:      "Lower = faster preference changes; Higher = more stable preferences",
			TradeOffs:   "Liveness vs Stability: Too low risks oscillation, too high risks getting stuck",
		},
		{
			Parameter:   "AlphaConfidence",
			Description: "Quorum threshold for confidence/finalization (safety threshold)",
			Formula:     "AlphaPreference ≤ AlphaConfidence ≤ K; typically 75-85% of K",
			MinValue:    "AlphaPreference",
			MaxValue:    "K",
			Typical:     "≈80% of K for strong safety",
			Impact:      "Higher = stronger finality guarantee; Lower = faster finalization",
			TradeOffs:   "Safety vs Speed: Higher values exponentially reduce probability of safety failure",
		},
		{
			Parameter:   "Beta",
			Description: "Number of consecutive successful rounds required for finalization",
			Formula:     "FinalityTime ≈ Beta × RoundLatency / ConcurrentRepolls",
			MinValue:    4,
			MaxValue:    100,
			Typical:     "8-20 for production, 4-8 for testing",
			Impact:      "Higher = stronger finality; Probability of error ≈ (1-AlphaConfidence/K)^Beta",
			TradeOffs:   "Security vs Latency: Each additional round exponentially improves safety",
		},
		{
			Parameter:   "ConcurrentRepolls",
			Description: "Number of consensus rounds that can be pipelined",
			Formula:     "1 ≤ ConcurrentRepolls ≤ Beta",
			MinValue:    1,
			MaxValue:    "Beta",
			Typical:     "4-20, often set equal to Beta for maximum throughput",
			Impact:      "Higher = better throughput via pipelining",
			TradeOffs:   "Throughput vs Complexity: More pipelining increases memory/CPU usage",
		},
		{
			Parameter:   "OptimalProcessing",
			Description: "Target number of consensus items to process in parallel",
			Formula:     "Based on available CPU cores and expected load",
			MinValue:    1,
			MaxValue:    100,
			Typical:     "10-32 depending on hardware",
			Impact:      "Affects CPU utilization and response time",
			TradeOffs:   "Parallelism vs Resource Usage: Too high causes context switching overhead",
		},
		{
			Parameter:   "MaxOutstandingItems",
			Description: "Maximum consensus items that can be in-flight simultaneously",
			Formula:     "Should be ≥ OptimalProcessing × expected_pipeline_depth",
			MinValue:    10,
			MaxValue:    10000,
			Typical:     "256-1024 for production",
			Impact:      "Caps memory usage and prevents overload",
			TradeOffs:   "Throughput vs Memory: Higher allows more parallelism but uses more RAM",
		},
		{
			Parameter:   "MaxItemProcessingTime",
			Description: "Timeout for processing a single consensus item",
			Formula:     "Should be >> expected finality time",
			MinValue:    "100ms",
			MaxValue:    "60s",
			Typical:     "5-10 seconds",
			Impact:      "Prevents stuck items from blocking progress",
			TradeOffs:   "Responsiveness vs Tolerance: Too low may timeout legitimate slow items",
		},
	}
}

// CalculateOptimalParameters suggests parameters based on network characteristics
type NetworkCharacteristics struct {
	TotalNodes           int
	ExpectedFailureRate  float64 // 0.0 to 1.0
	NetworkLatencyMs     int
	TargetFinalityMs     int
	TargetThroughputTPS  int
	IsProduction         bool
}

func CalculateOptimalParameters(nc NetworkCharacteristics) (*Parameters, string) {
	p := &Parameters{}
	var explanation []string
	
	// Step 1: Determine K (sample size)
	if nc.TotalNodes <= 30 {
		p.K = nc.TotalNodes
		explanation = append(explanation, 
			fmt.Sprintf("K=%d: Sampling all nodes for small network", p.K))
	} else {
		// Use sqrt(n) * 2 as a heuristic, capped at 50
		p.K = int(math.Sqrt(float64(nc.TotalNodes))) * 2
		if p.K > 50 {
			p.K = 50
		}
		explanation = append(explanation,
			fmt.Sprintf("K=%d: Using 2×√n sampling for %d nodes", p.K, nc.TotalNodes))
	}
	
	// Step 2: Calculate AlphaPreference based on expected failures
	// We want to tolerate ExpectedFailureRate in our sample
	minAlpha := (p.K / 2) + 1
	targetAlpha := int(float64(p.K) * (1 - nc.ExpectedFailureRate))
	if targetAlpha < minAlpha {
		targetAlpha = minAlpha
	}
	p.AlphaPreference = targetAlpha
	explanation = append(explanation,
		fmt.Sprintf("AlphaPreference=%d: Can tolerate %.0f%% failures", 
			p.AlphaPreference, nc.ExpectedFailureRate*100))
	
	// Step 3: Calculate AlphaConfidence for safety
	if nc.IsProduction {
		// Production: aim for 80-85% supermajority
		p.AlphaConfidence = int(float64(p.K) * 0.82)
		if p.AlphaConfidence < p.AlphaPreference {
			p.AlphaConfidence = p.AlphaPreference + 1
		}
	} else {
		// Testing: can use lower threshold
		p.AlphaConfidence = int(float64(p.K) * 0.75)
		if p.AlphaConfidence < p.AlphaPreference {
			p.AlphaConfidence = p.AlphaPreference
		}
	}
	explanation = append(explanation,
		fmt.Sprintf("AlphaConfidence=%d: %.0f%% supermajority for %s",
			p.AlphaConfidence, float64(p.AlphaConfidence)/float64(p.K)*100,
			map[bool]string{true: "production", false: "testing"}[nc.IsProduction]))
	
	// Step 4: Calculate Beta based on target finality
	roundTime := time.Duration(nc.NetworkLatencyMs) * time.Millisecond
	maxRounds := int(time.Duration(nc.TargetFinalityMs) * time.Millisecond / roundTime)
	
	if nc.IsProduction {
		// Production needs at least 8 rounds for security
		p.Beta = maxInt(8, maxRounds)
	} else {
		// Testing can use fewer rounds
		p.Beta = maxInt(4, maxRounds)
	}
	
	// Adjust Beta based on quorum strength
	quorumStrength := float64(p.AlphaConfidence) / float64(p.K)
	if quorumStrength < 0.75 {
		// Weaker quorum needs more rounds
		p.Beta = int(float64(p.Beta) * 1.5)
	}
	
	explanation = append(explanation,
		fmt.Sprintf("Beta=%d: Target finality %dms with %dms latency",
			p.Beta, nc.TargetFinalityMs, nc.NetworkLatencyMs))
	
	// Step 5: Set pipelining for throughput
	p.ConcurrentRepolls = p.Beta
	if p.ConcurrentRepolls > 20 {
		p.ConcurrentRepolls = 20 // Practical limit
	}
	explanation = append(explanation,
		fmt.Sprintf("ConcurrentRepolls=%d: Maximum pipelining", p.ConcurrentRepolls))
	
	// Step 6: Set processing parameters
	p.OptimalProcessing = 10
	if nc.TargetThroughputTPS > 1000 {
		p.OptimalProcessing = 32
	} else if nc.TargetThroughputTPS > 100 {
		p.OptimalProcessing = 20
	}
	
	p.MaxOutstandingItems = p.OptimalProcessing * 20
	if p.MaxOutstandingItems < 256 {
		p.MaxOutstandingItems = 256
	}
	
	// Step 7: Set timeout
	expectedFinality := p.CalculateExpectedFinality(nc.NetworkLatencyMs)
	p.MaxItemProcessingTime = expectedFinality * 10
	if p.MaxItemProcessingTime < 5*time.Second {
		p.MaxItemProcessingTime = 5 * time.Second
	}
	
	explanation = append(explanation,
		fmt.Sprintf("Timeout=%v: 10× expected finality", p.MaxItemProcessingTime))
	
	return p, fmt.Sprintf("Optimization reasoning:\n• %s", 
		strings.Join(explanation, "\n• "))
}

// ProbabilityAnalysis calculates various probability metrics
type ProbabilityAnalysis struct {
	SafetyFailureProbability    float64
	LivenessFailureProbability  float64
	ExpectedRoundsToFinality    float64
	ProbabilityOfDisagreement   float64
}

func AnalyzeProbabilities(p *Parameters, byzantineRatio float64) ProbabilityAnalysis {
	analysis := ProbabilityAnalysis{}
	
	// Probability that a single round produces incorrect supermajority
	// This happens when Byzantine nodes control AlphaConfidence votes
	singleRoundFailure := binomialProbability(p.K, p.AlphaConfidence, byzantineRatio)
	
	// Safety failure requires Beta consecutive bad rounds
	analysis.SafetyFailureProbability = math.Pow(singleRoundFailure, float64(p.Beta))
	
	// Liveness failure when honest nodes can't reach AlphaPreference
	honestRatio := 1 - byzantineRatio
	analysis.LivenessFailureProbability = 1 - binomialCDF(p.K, p.AlphaPreference-1, honestRatio)
	
	// Expected rounds accounts for network conditions
	successProbPerRound := binomialCDF(p.K, p.K-p.AlphaConfidence, byzantineRatio)
	if successProbPerRound > 0 {
		analysis.ExpectedRoundsToFinality = float64(p.Beta) / successProbPerRound
	} else {
		analysis.ExpectedRoundsToFinality = math.Inf(1)
	}
	
	// Probability two nodes disagree after protocol completes
	analysis.ProbabilityOfDisagreement = 2 * analysis.SafetyFailureProbability
	
	return analysis
}

// Helper functions
func binomialProbability(n, k int, p float64) float64 {
	return binomialCoeff(n, k) * math.Pow(p, float64(k)) * math.Pow(1-p, float64(n-k))
}

func binomialCDF(n, k int, p float64) float64 {
	sum := 0.0
	for i := 0; i <= k; i++ {
		sum += binomialProbability(n, i, p)
	}
	return sum
}

func binomialCoeff(n, k int) float64 {
	if k > n {
		return 0
	}
	if k == 0 || k == n {
		return 1
	}
	
	// Use logarithms to avoid overflow
	result := 0.0
	for i := 0; i < k; i++ {
		result += math.Log(float64(n-i)) - math.Log(float64(i+1))
	}
	return math.Exp(result)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}