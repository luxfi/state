// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package consensus

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// CheckerReport contains comprehensive analysis results
type CheckerReport struct {
	Parameters           *Parameters
	Warnings            []string
	LatencyAnalysis     LatencyAnalysis
	FailureProbabilities map[float64]FailureProb
	SafetyCutoff        float64
	LivenessAnalysis    LivenessAnalysis
	ThroughputAnalysis  ThroughputAnalysis
	Recommendations     []string
}

// LatencyAnalysis contains finality timing analysis
type LatencyAnalysis struct {
	TheoreticalMinimum   time.Duration
	ExpectedFinality     time.Duration
	WorstCaseFinality    time.Duration
	RoundTime            time.Duration
	PipelineEfficiency   float64
}

// FailureProb contains failure probability for a given adversarial stake
type FailureProb struct {
	AdversarialStake    float64
	PerRoundFailure     float64
	PerBlockFailure     float64
	ExpectedBlocksToFail float64
	YearsToFailure      float64
}

// LivenessAnalysis contains liveness metrics
type LivenessAnalysis struct {
	MinHonestNodesForProgress int
	MaxTolerableCrashes       int
	CrashTolerancePercent     float64
	PartitionTolerancePercent float64
}

// ThroughputAnalysis contains performance metrics
type ThroughputAnalysis struct {
	MaxTransactionsPerSecond  int
	MaxBlocksPerSecond        float64
	PipelineUtilization       float64
	ProcessingBottleneck      string
}

// RunChecker performs comprehensive parameter analysis
func RunChecker(p *Parameters, totalNodes int, networkLatencyMs int) *CheckerReport {
	report := &CheckerReport{
		Parameters:           p,
		Warnings:            []string{},
		FailureProbabilities: make(map[float64]FailureProb),
		Recommendations:     []string{},
	}

	// Validate parameters
	report.Warnings = append(report.Warnings, validateForChecker(p, totalNodes)...)

	// Analyze latency
	report.LatencyAnalysis = analyzeLatency(p, networkLatencyMs)

	// Analyze failure probabilities
	stakes := []float64{10, 20, 25, 30, 33, 40, 50}
	for _, stake := range stakes {
		report.FailureProbabilities[stake] = analyzeFailureProbability(p, stake/100)
	}

	// Find safety cutoff
	report.SafetyCutoff = findSafetyCutoff(p, 1e-9)

	// Analyze liveness
	report.LivenessAnalysis = analyzeLiveness(p, totalNodes)

	// Analyze throughput
	report.ThroughputAnalysis = analyzeThroughput(p, report.LatencyAnalysis)

	// Generate recommendations
	report.Recommendations = generateRecommendations(report, totalNodes)

	return report
}

func validateForChecker(p *Parameters, totalNodes int) []string {
	var warnings []string

	// Critical checks
	if p.K < 3 {
		warnings = append(warnings, "CRITICAL: K < 3 - insufficient sample size for consensus")
	}

	if p.AlphaPreference <= p.K/2 {
		warnings = append(warnings, fmt.Sprintf("CRITICAL: AlphaPreference (%d) ≤ 50%% of K (%d) - violates liveness requirement", 
			p.AlphaPreference, p.K))
	}

	if p.AlphaConfidence < p.AlphaPreference {
		warnings = append(warnings, fmt.Sprintf("CRITICAL: AlphaConfidence (%d) < AlphaPreference (%d) - invalid configuration", 
			p.AlphaConfidence, p.AlphaPreference))
	}

	if p.Beta < 1 {
		warnings = append(warnings, "CRITICAL: Beta < 1 - no rounds to achieve finality")
	}

	// Performance warnings
	if p.ConcurrentRepolls < p.Beta {
		warnings = append(warnings, fmt.Sprintf("PERFORMANCE: ConcurrentRepolls (%d) < Beta (%d) - suboptimal pipelining", 
			p.ConcurrentRepolls, p.Beta))
	}

	// Safety warnings
	ratio := float64(p.AlphaConfidence) / float64(p.K)
	if ratio > 0.9 {
		warnings = append(warnings, fmt.Sprintf("LIVENESS: High confidence quorum %.0f%% may hurt liveness under failures", ratio*100))
	} else if ratio < 0.67 {
		warnings = append(warnings, fmt.Sprintf("SAFETY: Low confidence quorum %.0f%% (< 67%%) risks Byzantine agreement safety", ratio*100))
	}

	// Network size warnings
	if p.K > totalNodes {
		warnings = append(warnings, fmt.Sprintf("INVALID: K (%d) > total nodes (%d)", p.K, totalNodes))
	} else if float64(p.K)/float64(totalNodes) < 0.5 && totalNodes <= 30 {
		warnings = append(warnings, fmt.Sprintf("SUBOPTIMAL: Only sampling %d/%d (%.0f%%) nodes in small network", 
			p.K, totalNodes, float64(p.K)/float64(totalNodes)*100))
	}

	return warnings
}

func analyzeLatency(p *Parameters, networkLatencyMs int) LatencyAnalysis {
	rtt := time.Duration(networkLatencyMs) * time.Millisecond
	roundTime := rtt + 10*time.Millisecond // Add processing overhead

	// Theoretical minimum with perfect pipelining
	theoretical := roundTime

	// Expected with pipelining
	pipelineDepth := float64(p.ConcurrentRepolls)
	if pipelineDepth > float64(p.Beta) {
		pipelineDepth = float64(p.Beta)
	}
	pipelineEfficiency := pipelineDepth / float64(p.Beta)
	expected := time.Duration(float64(p.Beta*int(roundTime)) / pipelineDepth)

	// Worst case (no pipelining, network delays)
	worstCase := time.Duration(p.Beta) * roundTime * 2

	return LatencyAnalysis{
		TheoreticalMinimum: theoretical,
		ExpectedFinality:   expected,
		WorstCaseFinality:  worstCase,
		RoundTime:          roundTime,
		PipelineEfficiency: pipelineEfficiency,
	}
}

func analyzeFailureProbability(p *Parameters, adversarialRatio float64) FailureProb {
	// Per-round failure: probability adversary controls ≥ AlphaConfidence votes
	perRound := 0.0
	for i := p.AlphaConfidence; i <= p.K; i++ {
		prob := binomialProbability(p.K, i, adversarialRatio)
		perRound += prob
	}

	// Per-block failure: need Beta consecutive bad rounds
	perBlock := math.Pow(perRound, float64(p.Beta))

	// Expected blocks to failure
	blocksToFail := math.Inf(1)
	if perBlock > 0 {
		blocksToFail = 1 / perBlock
	}

	// Convert to years (assuming 1 block per second)
	yearsToFail := blocksToFail / (365.25 * 24 * 60 * 60)

	return FailureProb{
		AdversarialStake:     adversarialRatio * 100,
		PerRoundFailure:      perRound,
		PerBlockFailure:      perBlock,
		ExpectedBlocksToFail: blocksToFail,
		YearsToFailure:       yearsToFail,
	}
}

func findSafetyCutoff(p *Parameters, targetEpsilon float64) float64 {
	low, high := 0.0, 1.0
	
	// Binary search for the cutoff
	for i := 0; i < 60; i++ {
		mid := (low + high) / 2
		fp := analyzeFailureProbability(p, mid)
		if fp.PerBlockFailure > targetEpsilon {
			high = mid
		} else {
			low = mid
		}
	}

	return high * 100 // Return as percentage
}

func analyzeLiveness(p *Parameters, totalNodes int) LivenessAnalysis {
	// Minimum honest nodes needed for progress
	minHonest := p.AlphaPreference
	maxCrashes := p.K - p.AlphaPreference

	// Calculate percentages
	crashTolerance := float64(maxCrashes) / float64(p.K) * 100
	
	// Partition tolerance (nodes that can be unreachable)
	partitionTolerance := float64(totalNodes-p.AlphaPreference) / float64(totalNodes) * 100

	return LivenessAnalysis{
		MinHonestNodesForProgress: minHonest,
		MaxTolerableCrashes:       maxCrashes,
		CrashTolerancePercent:     crashTolerance,
		PartitionTolerancePercent: partitionTolerance,
	}
}

func analyzeThroughput(p *Parameters, latency LatencyAnalysis) ThroughputAnalysis {
	// Blocks per second based on finality time
	blocksPerSecond := 1.0 / latency.ExpectedFinality.Seconds()

	// Transactions per second (assuming 100 tx per block)
	txPerBlock := 100
	maxTPS := int(blocksPerSecond * float64(txPerBlock))

	// Pipeline utilization
	utilization := float64(p.ConcurrentRepolls) / float64(p.Beta)
	if utilization > 1 {
		utilization = 1
	}

	// Identify bottleneck
	bottleneck := "Network latency"
	if p.ConcurrentRepolls < p.Beta {
		bottleneck = "Pipeline depth"
	} else if p.MaxOutstandingItems < p.OptimalProcessing*10 {
		bottleneck = "Outstanding item limit"
	}

	return ThroughputAnalysis{
		MaxTransactionsPerSecond: maxTPS,
		MaxBlocksPerSecond:       blocksPerSecond,
		PipelineUtilization:      utilization,
		ProcessingBottleneck:     bottleneck,
	}
}

func generateRecommendations(report *CheckerReport, totalNodes int) []string {
	var recs []string
	p := report.Parameters

	// Pipelining optimization
	if p.ConcurrentRepolls < p.Beta {
		recs = append(recs, fmt.Sprintf("Increase ConcurrentRepolls to %d to maximize throughput", p.Beta))
	}

	// Safety recommendations
	if report.SafetyCutoff < 25 {
		recs = append(recs, fmt.Sprintf("Safety cutoff is %.1f%% - consider increasing AlphaConfidence or Beta", report.SafetyCutoff))
	}

	// Liveness recommendations
	if report.LivenessAnalysis.CrashTolerancePercent < 20 {
		recs = append(recs, "Low crash tolerance - consider reducing AlphaPreference for better liveness")
	}

	// Latency recommendations
	if report.LatencyAnalysis.ExpectedFinality > time.Second {
		if p.Beta > 10 {
			recs = append(recs, "High finality latency - consider reducing Beta if safety margins allow")
		}
		if p.ConcurrentRepolls < p.Beta {
			recs = append(recs, "Improve latency by increasing ConcurrentRepolls for better pipelining")
		}
	}

	// Network-specific recommendations
	if totalNodes <= 10 && p.K < totalNodes {
		recs = append(recs, fmt.Sprintf("Small network - consider setting K=%d to sample all nodes", totalNodes))
	}

	return recs
}

// FormatCheckerReport generates a detailed human-readable report
func FormatCheckerReport(report *CheckerReport, totalNodes int) string {
	var b strings.Builder

	b.WriteString("╔════════════════════════════════════════════════════════════╗\n")
	b.WriteString("║           LUX CONSENSUS PARAMETER ANALYSIS REPORT          ║\n")
	b.WriteString("╚════════════════════════════════════════════════════════════╝\n\n")

	// Parameters summary
	b.WriteString("📊 PARAMETER CONFIGURATION\n")
	b.WriteString("─────────────────────────\n")
	p := report.Parameters
	b.WriteString(fmt.Sprintf("• Sample Size (K): %d nodes\n", p.K))
	b.WriteString(fmt.Sprintf("• Preference Quorum: %d/%d (%.1f%%)\n", 
		p.AlphaPreference, p.K, float64(p.AlphaPreference)/float64(p.K)*100))
	b.WriteString(fmt.Sprintf("• Confidence Quorum: %d/%d (%.1f%%)\n", 
		p.AlphaConfidence, p.K, float64(p.AlphaConfidence)/float64(p.K)*100))
	b.WriteString(fmt.Sprintf("• Finalization Rounds (Beta): %d\n", p.Beta))
	b.WriteString(fmt.Sprintf("• Pipeline Depth: %d\n", p.ConcurrentRepolls))

	// Warnings
	if len(report.Warnings) > 0 {
		b.WriteString("\n⚠️  WARNINGS\n")
		b.WriteString("────────────\n")
		for _, w := range report.Warnings {
			b.WriteString(fmt.Sprintf("• %s\n", w))
		}
	}

	// Latency analysis
	b.WriteString("\n⏱️  FINALITY TIMING ANALYSIS\n")
	b.WriteString("────────────────────────────\n")
	la := report.LatencyAnalysis
	b.WriteString(fmt.Sprintf("• Expected Finality: %v\n", la.ExpectedFinality))
	b.WriteString(fmt.Sprintf("• Theoretical Minimum: %v\n", la.TheoreticalMinimum))
	b.WriteString(fmt.Sprintf("• Worst Case: %v\n", la.WorstCaseFinality))
	b.WriteString(fmt.Sprintf("• Pipeline Efficiency: %.0f%%\n", la.PipelineEfficiency*100))

	// Mathematical explanation
	b.WriteString("\n📐 MATHEMATICAL BREAKDOWN\n")
	b.WriteString("─────────────────────────\n")
	b.WriteString(fmt.Sprintf("Finality = (Beta × RoundTime) / PipelineDepth\n"))
	b.WriteString(fmt.Sprintf("         = (%d × %v) / %d\n", p.Beta, la.RoundTime, p.ConcurrentRepolls))
	b.WriteString(fmt.Sprintf("         = %v\n", la.ExpectedFinality))

	// Failure probability analysis
	b.WriteString("\n🔒 SECURITY ANALYSIS (Failure Probabilities)\n")
	b.WriteString("────────────────────────────────────────────\n")
	b.WriteString("Adversary | Per-Round | Per-Block | Expected Time to Failure\n")
	b.WriteString("----------|-----------|-----------|-------------------------\n")
	
	// Sort stakes for display
	stakes := []float64{10, 20, 25, 30, 33, 40, 50}
	for _, stake := range stakes {
		if fp, exists := report.FailureProbabilities[stake]; exists {
			timeStr := "Never"
			if fp.YearsToFailure < 1e15 {
				if fp.YearsToFailure > 1 {
					timeStr = fmt.Sprintf("%.1f years", fp.YearsToFailure)
				} else if fp.YearsToFailure*365 > 1 {
					timeStr = fmt.Sprintf("%.1f days", fp.YearsToFailure*365)
				} else {
					timeStr = fmt.Sprintf("%.1f hours", fp.YearsToFailure*365*24)
				}
			}
			b.WriteString(fmt.Sprintf("   %2.0f%%   | %.2e | %.2e | %s\n",
				stake, fp.PerRoundFailure, fp.PerBlockFailure, timeStr))
		}
	}

	b.WriteString(fmt.Sprintf("\n🎯 Safety Cutoff (ε ≤ 10⁻⁹): %.1f%% adversarial stake\n", report.SafetyCutoff))

	// Liveness analysis
	b.WriteString("\n💚 LIVENESS ANALYSIS\n")
	b.WriteString("────────────────────\n")
	la2 := report.LivenessAnalysis
	b.WriteString(fmt.Sprintf("• Can tolerate %d/%d (%.0f%%) crashed nodes\n", 
		la2.MaxTolerableCrashes, p.K, la2.CrashTolerancePercent))
	b.WriteString(fmt.Sprintf("• Network partition tolerance: %.0f%% unreachable\n", 
		la2.PartitionTolerancePercent))
	b.WriteString(fmt.Sprintf("• Minimum honest nodes for progress: %d\n", 
		la2.MinHonestNodesForProgress))

	// Throughput analysis
	b.WriteString("\n🚀 THROUGHPUT ANALYSIS\n")
	b.WriteString("──────────────────────\n")
	ta := report.ThroughputAnalysis
	b.WriteString(fmt.Sprintf("• Max Blocks/Second: %.2f\n", ta.MaxBlocksPerSecond))
	b.WriteString(fmt.Sprintf("• Max Transactions/Second: ~%d\n", ta.MaxTransactionsPerSecond))
	b.WriteString(fmt.Sprintf("• Pipeline Utilization: %.0f%%\n", ta.PipelineUtilization*100))
	b.WriteString(fmt.Sprintf("• Bottleneck: %s\n", ta.ProcessingBottleneck))

	// Recommendations
	if len(report.Recommendations) > 0 {
		b.WriteString("\n💡 RECOMMENDATIONS\n")
		b.WriteString("──────────────────\n")
		for i, rec := range report.Recommendations {
			b.WriteString(fmt.Sprintf("%d. %s\n", i+1, rec))
		}
	}

	// How it works section
	b.WriteString("\n📚 HOW CONSENSUS WORKS WITH THESE SETTINGS\n")
	b.WriteString("──────────────────────────────────────────\n")
	b.WriteString(fmt.Sprintf(
`1. SAMPLING: Each node queries %d random validators per round

2. PREFERENCE: If ≥%d validators agree, node updates its preference
   - This requires only %.0f%% agreement, allowing quick convergence
   - Can progress even if %d nodes crash or are slow

3. CONFIDENCE: If ≥%d validators agree, round counts toward finality
   - This %.0f%% supermajority provides strong safety guarantees
   - Adversary needs >%d nodes in sample to cause disagreement

4. FINALIZATION: After %d consecutive successful rounds
   - Probability of finalization error: (adversary_control)^%d
   - With %.0f%% adversary: %.2e chance of failure

5. PIPELINING: %d rounds execute concurrently
   - Reduces latency from %v to %v
   - Achieves %.0f%% of theoretical maximum throughput`,
		p.K,
		p.AlphaPreference,
		float64(p.AlphaPreference)/float64(p.K)*100,
		p.K-p.AlphaPreference,
		p.AlphaConfidence,
		float64(p.AlphaConfidence)/float64(p.K)*100,
		p.K-p.AlphaConfidence,
		p.Beta,
		p.Beta,
		report.SafetyCutoff,
		report.FailureProbabilities[report.SafetyCutoff].PerBlockFailure,
		p.ConcurrentRepolls,
		time.Duration(p.Beta)*la.RoundTime,
		la.ExpectedFinality,
		la.PipelineEfficiency*100))

	b.WriteString("\n\n═══════════════════════════════════════════════════════════════\n")

	return b.String()
}