// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/luxfi/genesis/consensus"
)

func main() {
	var (
		preset      = flag.String("preset", "", "Use preset configuration: mainnet, testnet, local")
		nodeCount   = flag.Int("nodes", 0, "Number of nodes in the network")
		k           = flag.Int("k", 0, "Sample size")
		alphaPref   = flag.Int("alpha-pref", 0, "Preference quorum threshold")
		alphaConf   = flag.Int("alpha-conf", 0, "Confidence quorum threshold")
		beta        = flag.Int("beta", 0, "Consecutive rounds threshold")
		concurrent  = flag.Int("concurrent", 0, "Concurrent repolls")
		optimize    = flag.String("optimize", "", "Optimize for: latency, security, throughput")
		output      = flag.String("output", "", "Output file for parameters (JSON)")
		summary     = flag.Bool("summary", false, "Show parameter summary")
		validate    = flag.String("validate", "", "Validate parameters from JSON file")
		targetTime  = flag.Duration("target-finality", 0*time.Second, "Target finality time")
		networkLat  = flag.Int("network-latency", 50, "Expected network latency in ms")
	)

	flag.Parse()

	// Handle validation mode
	if *validate != "" {
		if err := validateFile(*validate); err != nil {
			fmt.Fprintf(os.Stderr, "Validation failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Parameters are valid!")
		return
	}

	// Build parameters
	builder := consensus.NewBuilder()

	// Apply preset if specified
	if *preset != "" {
		b, err := builder.FromPreset(consensus.NetworkType(*preset))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading preset: %v\n", err)
			os.Exit(1)
		}
		builder = b
	}

	// Apply node count optimization
	if *nodeCount > 0 {
		builder = builder.ForNodeCount(*nodeCount)
	}

	// Apply manual parameters
	if *k > 0 {
		builder = builder.WithSampleSize(*k)
	}
	if *alphaPref > 0 || *alphaConf > 0 {
		// Use existing values if not specified
		params, _ := builder.Build()
		pref := *alphaPref
		if pref == 0 {
			pref = params.AlphaPreference
		}
		conf := *alphaConf
		if conf == 0 {
			conf = params.AlphaConfidence
		}
		builder = builder.WithQuorums(pref, conf)
	}
	if *beta > 0 {
		builder = builder.WithBeta(*beta)
	}
	if *concurrent > 0 {
		builder = builder.WithConcurrentRepolls(*concurrent)
	}

	// Apply target finality
	if *targetTime > 0 {
		builder = builder.WithTargetFinality(*targetTime, *networkLat)
	}

	// Apply optimization
	switch *optimize {
	case "latency":
		builder = builder.OptimizeForLatency()
	case "security":
		builder = builder.OptimizeForSecurity()
	case "throughput":
		builder = builder.OptimizeForThroughput()
	}

	// Build final parameters
	params, err := builder.Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error building parameters: %v\n", err)
		os.Exit(1)
	}

	// Output results
	if *output != "" {
		data, err := params.ToJSON()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
			os.Exit(1)
		}
		if err := ioutil.WriteFile(*output, data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Parameters written to %s\n", *output)
	} else {
		// Print JSON to stdout
		data, _ := params.ToJSON()
		fmt.Println(string(data))
	}

	// Show summary if requested
	if *summary {
		fmt.Println("\n" + params.Summary())
	}
}

func validateFile(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	params, err := consensus.FromJSON(data)
	if err != nil {
		return fmt.Errorf("parsing JSON: %w", err)
	}

	if err := params.Validate(); err != nil {
		return fmt.Errorf("validation: %w", err)
	}

	fmt.Println(params.Summary())
	return nil
}