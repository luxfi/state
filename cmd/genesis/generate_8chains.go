// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package main

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

// generate8ChainsCmd creates the command for generating 8-chain genesis
func generate8ChainsCmd() *cobra.Command {
	var (
		numValidators   int
		initialStake    uint64
		stakingDuration time.Duration
		aiAgentCount    int
		bridgeThreshold int
		mpcParticipants int
		zkCircuitCount  int
		cpuAffinity     bool
	)

	cmd := &cobra.Command{
		Use:   "8chains",
		Short: "Generate genesis files for all 8 chains (P, C, X, A, B, M, Q, Z)",
		Long: `Generate genesis files for all 8 chains with proper configuration.
		
This command creates genesis files for:
- P-Chain (Platform): Core validation and subnet management
- C-Chain (EVM): Ethereum Virtual Machine compatible smart contracts
- X-Chain (Exchange): Asset creation and atomic swaps
- A-Chain (AI): AI agent coordination and GPU compute
- B-Chain (Bridge): Cross-chain bridge operations
- M-Chain (MPC): Multi-party computation for secure operations
- Q-Chain (Quantum): Quantum-safe cryptography
- Z-Chain (ZK): Zero-knowledge proof circuits

The generated genesis files will be placed in the output directory
with the proper structure expected by luxd.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			outputPath, _ := cmd.Flags().GetString("output")
			network, _ := cmd.Flags().GetString("network")
			
			fmt.Printf("🚀 Generating 8-chain genesis for %s network\n", network)
			fmt.Printf("📁 Output directory: %s\n", outputPath)
			fmt.Printf("🔢 Validators: %d\n", numValidators)
			fmt.Printf("💰 Initial stake per validator: %d LUX\n", initialStake)
			fmt.Printf("⏱️  Staking duration: %s\n", stakingDuration)
			
			// Create output directory
			if err := os.MkdirAll(outputPath, 0755); err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}
			
			// Generate validators
			validators := generateValidators(numValidators, initialStake, stakingDuration)
			
			// Generate P-Chain genesis (includes all chain configurations)
			if err := generate8ChainsPChainGenesis(outputPath, network, validators); err != nil {
				return fmt.Errorf("failed to generate P-Chain genesis: %w", err)
			}
			
			// Generate C-Chain genesis
			if err := generate8ChainsCChainGenesis(outputPath, network); err != nil {
				return fmt.Errorf("failed to generate C-Chain genesis: %w", err)
			}
			
			// Generate X-Chain genesis
			if err := generate8ChainsXChainGenesis(outputPath, network); err != nil {
				return fmt.Errorf("failed to generate X-Chain genesis: %w", err)
			}
			
			// Generate custom chain genesis files
			customChains := map[string]func(string, map[string]interface{}) error{
				"A": func(path string, params map[string]interface{}) error {
					return generateAChainGenesis(path, params["aiAgentCount"].(int))
				},
				"B": func(path string, params map[string]interface{}) error {
					return generateBChainGenesis(path, params["bridgeThreshold"].(int), params["mpcParticipants"].(int))
				},
				"M": func(path string, params map[string]interface{}) error {
					return generateMChainGenesis(path, params["mpcParticipants"].(int), params["bridgeThreshold"].(int))
				},
				"Q": func(path string, params map[string]interface{}) error {
					return generateQChainGenesis(path)
				},
				"Z": func(path string, params map[string]interface{}) error {
					return generateZChainGenesis(path, params["zkCircuitCount"].(int))
				},
			}
			
			params := map[string]interface{}{
				"aiAgentCount":    aiAgentCount,
				"bridgeThreshold": bridgeThreshold,
				"mpcParticipants": mpcParticipants,
				"zkCircuitCount":  zkCircuitCount,
			}
			
			for chain, genFunc := range customChains {
				chainPath := filepath.Join(outputPath, chain)
				if err := os.MkdirAll(chainPath, 0755); err != nil {
					return fmt.Errorf("failed to create %s-Chain directory: %w", chain, err)
				}
				
				if err := genFunc(chainPath, params); err != nil {
					return fmt.Errorf("failed to generate %s-Chain genesis: %w", chain, err)
				}
			}
			
			// Generate CPU affinity configuration if requested
			if cpuAffinity {
				if err := generateCPUAffinityConfig(outputPath); err != nil {
					return fmt.Errorf("failed to generate CPU affinity config: %w", err)
				}
			}
			
			// Generate bootstrap configuration
			if err := generateBootstrapConfig(outputPath, network); err != nil {
				return fmt.Errorf("failed to generate bootstrap config: %w", err)
			}
			
			fmt.Println("\n✅ Successfully generated 8-chain genesis configuration!")
			fmt.Printf("📂 Genesis files written to: %s\n", outputPath)
			fmt.Println("\n🚀 To launch the network, run:")
			fmt.Printf("   luxd --genesis-config=%s\n", outputPath)
			
			return nil
		},
	}
	
	// Add flags
	cmd.Flags().IntVar(&numValidators, "validators", 8, "Number of validators")
	cmd.Flags().Uint64Var(&initialStake, "stake", 2000, "Initial stake per validator in LUX")
	cmd.Flags().DurationVar(&stakingDuration, "staking-duration", 365*24*time.Hour, "Staking duration")
	cmd.Flags().IntVar(&aiAgentCount, "ai-agents", 10, "Number of AI agents for A-Chain")
	cmd.Flags().IntVar(&bridgeThreshold, "bridge-threshold", 5, "Threshold for bridge operations")
	cmd.Flags().IntVar(&mpcParticipants, "mpc-participants", 8, "Number of MPC participants")
	cmd.Flags().IntVar(&zkCircuitCount, "zk-circuits", 5, "Number of ZK circuits")
	cmd.Flags().BoolVar(&cpuAffinity, "cpu-affinity", false, "Generate CPU affinity configuration")
	
	return cmd
}

func generateValidators(count int, stake uint64, duration time.Duration) []map[string]interface{} {
	validators := make([]map[string]interface{}, count)
	startTime := time.Now().Unix()
	endTime := time.Now().Add(duration).Unix()
	
	for i := 0; i < count; i++ {
		validators[i] = map[string]interface{}{
			"nodeID": fmt.Sprintf("NodeID-%d%s", i+1, generateNodeIDSuffix()),
			"startTime": startTime,
			"endTime": endTime,
			"stakeAmount": stake * 1e9, // Convert to nLUX
		}
	}
	
	return validators
}

func generate8ChainsPChainGenesis(outputPath, network string, validators []map[string]interface{}) error {
	fmt.Println("📝 Generating P-Chain genesis with 8-chain configuration...")
	
	pPath := filepath.Join(outputPath, "P")
	if err := os.MkdirAll(pPath, 0755); err != nil {
		return err
	}
	
	// Define all chains including custom ones
	chains := []map[string]interface{}{
		// Standard chains are created automatically
		// Custom chains need subnet configurations
		{
			"subnetID": generateSubnetID("aivm"),
			"chainName": "A-Chain",
			"vmID": "juFxSrbCM4wszxddKepj1GWwmrn9YgN1g4n3VUWPpRo9JjERA", // AIVM ID
		},
		{
			"subnetID": generateSubnetID("bridgevm"),
			"chainName": "B-Chain", 
			"vmID": "kMhHABHM8j4bH94MCc4rsTNdo5E9En37MMyiujk4WdNxgXFsY", // BridgeVM ID
		},
		{
			"subnetID": generateSubnetID("mpcvm"),
			"chainName": "M-Chain",
			"vmID": "qCURact1n41FcoNBch8iMVBwc9AWie48D118ZNJ5tBdWrvryS", // MPCVM ID
		},
		{
			"subnetID": generateSubnetID("quantumvm"),
			"chainName": "Q-Chain",
			"vmID": "ry9Sg8rZdT26iEKvJDmC2wkESs4SDKgZEhk5BgLSwg1EpcNug", // QuantumVM ID
		},
		{
			"subnetID": generateSubnetID("zkvm"),
			"chainName": "Z-Chain",
			"vmID": "vv3qPfyTVXZ5ArRZA9Jh4hbYDTBe43f7sgQg4CHfNg1rnnvX9", // ZKVM ID
		},
	}
	
	genesis := map[string]interface{}{
		"networkID": getNetworkID(network),
		"allocations": []interface{}{},
		"startTime": time.Now().Unix(),
		"initialStakeDuration": 365 * 24 * 60 * 60, // 1 year in seconds
		"initialStakeDurationOffset": 5 * 60, // 5 minutes
		"initialStakedFunds": validators,
		"initialStakers": validators,
		"cChainGenesis": "{}", // Will be replaced with actual C-Chain genesis
		"chains": chains,
		"message": "8-Chain Lux Network Genesis",
	}
	
	data, err := json.MarshalIndent(genesis, "", "  ")
	if err != nil {
		return err
	}
	
	genesisFile := filepath.Join(pPath, "genesis.json")
	return os.WriteFile(genesisFile, data, 0644)
}

func generate8ChainsCChainGenesis(outputPath, network string) error {
	fmt.Println("📝 Generating C-Chain genesis...")
	
	cPath := filepath.Join(outputPath, "C")
	if err := os.MkdirAll(cPath, 0755); err != nil {
		return err
	}
	
	chainID := getChainID(network)
	
	genesis := map[string]interface{}{
		"config": map[string]interface{}{
			"chainId": chainID,
			"homesteadBlock": 0,
			"eip150Block": 0,
			"eip150Hash": "0x2086799aeebeae135c246c65021c82b4e15a2c451340993aacfd2751886514f0",
			"eip155Block": 0,
			"eip158Block": 0,
			"byzantiumBlock": 0,
			"constantinopleBlock": 0,
			"petersburgBlock": 0,
			"istanbulBlock": 0,
			"muirGlacierBlock": 0,
			"subnetEVMTimestamp": 0,
			"feeConfig": map[string]interface{}{
				"gasLimit": 15000000,
				"targetBlockRate": 2,
				"minBaseFee": 25000000000,
				"targetGas": 15000000,
				"baseFeeChangeDenominator": 36,
				"minBlockGasCost": 0,
				"maxBlockGasCost": 1000000,
				"blockGasCostStep": 200000,
			},
		},
		"nonce": "0x0",
		"timestamp": fmt.Sprintf("0x%x", time.Now().Unix()),
		"extraData": "0x00",
		"gasLimit": "0xe4e1c0",
		"difficulty": "0x0",
		"mixHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
		"coinbase": "0x0000000000000000000000000000000000000000",
		"alloc": map[string]interface{}{
			// Pre-funded accounts for testing
			"0x1000000000000000000000000000000000000001": map[string]interface{}{
				"balance": "0x52b7d2dcc80cd2e4000000", // 100M tokens
			},
		},
		"number": "0x0",
		"gasUsed": "0x0",
		"parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
	}
	
	data, err := json.MarshalIndent(genesis, "", "  ")
	if err != nil {
		return err
	}
	
	genesisFile := filepath.Join(cPath, "genesis.json")
	return os.WriteFile(genesisFile, data, 0644)
}

func generate8ChainsXChainGenesis(outputPath, network string) error {
	fmt.Println("📝 Generating X-Chain genesis...")
	
	xPath := filepath.Join(outputPath, "X")
	if err := os.MkdirAll(xPath, 0755); err != nil {
		return err
	}
	
	genesis := map[string]interface{}{
		"networkID": getNetworkID(network),
		"allocations": []interface{}{},
		"startTime": time.Now().Unix(),
		"initialSupply": 720000000000000000, // 720M LUX
		"message": "X-Chain Genesis",
	}
	
	data, err := json.MarshalIndent(genesis, "", "  ")
	if err != nil {
		return err
	}
	
	genesisFile := filepath.Join(xPath, "genesis.json")
	return os.WriteFile(genesisFile, data, 0644)
}

// Custom chain genesis generators

func generateAChainGenesis(outputPath string, agentCount int) error {
	fmt.Println("📝 Generating A-Chain (AI) genesis...")
	
	agents := make([]map[string]interface{}, agentCount)
	for i := 0; i < agentCount; i++ {
		agents[i] = map[string]interface{}{
			"id": fmt.Sprintf("agent-%03d", i),
			"name": fmt.Sprintf("AI Agent %d", i),
			"modelType": "llm-7b",
			"gpuProvider": fmt.Sprintf("gpu-pool-%d", i%3),
			"reputation": 100,
			"stake": 1000000000000, // 1000 LUX in nLUX
		}
	}
	
	genesis := map[string]interface{}{
		"config": map[string]interface{}{
			"blockTime": 2000, // 2 seconds
			"minGasPrice": 1000000000,
			"maxGasLimit": 15000000,
			"targetGasUsage": 10000000,
			"minAgentStake": 1000000000000,
			"taskTimeout": 300,
			"reputationDecay": 0.99,
		},
		"agents": agents,
		"gpuProviders": []map[string]interface{}{
			{
				"id": "gpu-pool-0",
				"capacity": 1000,
				"type": "A100",
				"location": "us-east-1",
			},
			{
				"id": "gpu-pool-1",
				"capacity": 500,
				"type": "H100",
				"location": "us-west-2",
			},
			{
				"id": "gpu-pool-2",
				"capacity": 750,
				"type": "RTX4090",
				"location": "eu-central-1",
			},
		},
		"modelRegistry": []map[string]interface{}{
			{
				"id": "llm-7b",
				"name": "LLM 7B Base",
				"parameters": 7000000000,
				"gpuMemory": 16,
			},
		},
	}
	
	data, err := json.MarshalIndent(genesis, "", "  ")
	if err != nil {
		return err
	}
	
	genesisFile := filepath.Join(outputPath, "genesis.json")
	return os.WriteFile(genesisFile, data, 0644)
}

func generateBChainGenesis(outputPath string, threshold, mpcNodes int) error {
	fmt.Println("📝 Generating B-Chain (Bridge) genesis...")
	
	nodes := make([]map[string]interface{}, mpcNodes)
	for i := 0; i < mpcNodes; i++ {
		nodes[i] = map[string]interface{}{
			"id": fmt.Sprintf("mpc-node-%d", i),
			"index": i,
			"publicKey": fmt.Sprintf("0x%064x", i),
			"stake": 1000000000000,
			"reputation": 100,
			"endpoint": fmt.Sprintf("mpc-node-%d:9651", i),
		}
	}
	
	genesis := map[string]interface{}{
		"config": map[string]interface{}{
			"blockTime": 3000,
			"minSignatures": threshold,
			"bridgeTimeout": 600,
			"maxBridgeAmount": 1000000000000000,
			"bridgeFee": 0.001,
		},
		"bridges": []map[string]interface{}{
			{
				"id": "eth-mainnet",
				"targetChain": "ethereum",
				"chainId": 1,
				"threshold": threshold,
				"address": "0x0000000000000000000000000000000000000000",
				"status": "active",
			},
			{
				"id": "bsc-mainnet",
				"targetChain": "bsc",
				"chainId": 56,
				"threshold": threshold,
				"address": "0x0000000000000000000000000000000000000000",
				"status": "active",
			},
		},
		"mpcNodes": nodes,
		"treasury": map[string]interface{}{
			"address": "0x0000000000000000000000000000000000000001",
			"balance": 10000000000000,
		},
	}
	
	data, err := json.MarshalIndent(genesis, "", "  ")
	if err != nil {
		return err
	}
	
	genesisFile := filepath.Join(outputPath, "genesis.json")
	return os.WriteFile(genesisFile, data, 0644)
}

func generateMChainGenesis(outputPath string, participants, threshold int) error {
	fmt.Println("📝 Generating M-Chain (MPC) genesis...")
	
	nodes := make([]map[string]interface{}, participants)
	for i := 0; i < participants; i++ {
		nodes[i] = map[string]interface{}{
			"id": fmt.Sprintf("mpc-%03d", i),
			"publicKey": fmt.Sprintf("0x%064x", i),
			"stake": 1000000000000,
		}
	}
	
	genesis := map[string]interface{}{
		"config": map[string]interface{}{
			"blockTime": 2000,
			"mpcThreshold": threshold,
			"sessionLength": 100,
			"keyGenTimeout": 60,
			"signTimeout": 30,
			"minParticipants": threshold,
			"maxParticipants": participants,
			"slashingPenalty": 100000000000,
			"sessionReward": 10000000000,
		},
		"initialNodes": nodes,
		"protocols": []map[string]interface{}{
			{
				"id": "gg20",
				"name": "Gennaro-Goldfeder 2020",
				"threshold": threshold,
				"supported": true,
			},
			{
				"id": "frost",
				"name": "FROST",
				"threshold": threshold,
				"supported": true,
			},
		},
		"sessions": []interface{}{},
	}
	
	data, err := json.MarshalIndent(genesis, "", "  ")
	if err != nil {
		return err
	}
	
	genesisFile := filepath.Join(outputPath, "genesis.json")
	return os.WriteFile(genesisFile, data, 0644)
}

func generateQChainGenesis(outputPath string) error {
	fmt.Println("📝 Generating Q-Chain (Quantum) genesis...")
	
	genesis := map[string]interface{}{
		"config": map[string]interface{}{
			"blockTime": 2000,
			"signatureAlgo": "sphincs+",
			"hashFunction": "sha3-256",
			"securityLevel": 256,
			"publicKeySize": 64,
			"signatureSize": 49856,
			"migrationStart": time.Now().Add(365 * 24 * time.Hour).Unix(),
		},
		"validators": []interface{}{},
		"quantumAlgorithms": []map[string]interface{}{
			{
				"id": "sphincs+",
				"name": "SPHINCS+",
				"type": "signature",
				"securityLevel": 256,
				"status": "active",
			},
			{
				"id": "dilithium",
				"name": "CRYSTALS-Dilithium",
				"type": "signature", 
				"securityLevel": 256,
				"status": "experimental",
			},
			{
				"id": "kyber",
				"name": "CRYSTALS-Kyber",
				"type": "kem",
				"securityLevel": 256,
				"status": "experimental",
			},
		},
		"migrationPlan": map[string]interface{}{
			"phase1Start": time.Now().Add(180 * 24 * time.Hour).Unix(),
			"phase2Start": time.Now().Add(365 * 24 * time.Hour).Unix(),
			"mandatory": time.Now().Add(730 * 24 * time.Hour).Unix(),
		},
	}
	
	data, err := json.MarshalIndent(genesis, "", "  ")
	if err != nil {
		return err
	}
	
	genesisFile := filepath.Join(outputPath, "genesis.json")
	return os.WriteFile(genesisFile, data, 0644)
}

func generateZChainGenesis(outputPath string, circuitCount int) error {
	fmt.Println("📝 Generating Z-Chain (ZK) genesis...")
	
	circuits := make([]map[string]interface{}, circuitCount)
	circuitTypes := []string{"transfer", "mint", "burn", "swap", "stake"}
	
	for i := 0; i < circuitCount; i++ {
		circuits[i] = map[string]interface{}{
			"id": fmt.Sprintf("circuit-%03d", i),
			"name": fmt.Sprintf("ZK %s Circuit", circuitTypes[i%len(circuitTypes)]),
			"type": circuitTypes[i%len(circuitTypes)],
			"proofSystem": "plonk",
			"constraintSize": 100000 + i*10000,
			"setupComplete": true,
			"srsHash": fmt.Sprintf("0x%064x", i),
		}
	}
	
	genesis := map[string]interface{}{
		"config": map[string]interface{}{
			"blockTime": 4000,
			"proofSystem": "plonk",
			"curveType": "bn254",
			"maxProofSize": 2048,
			"maxConstraints": 1000000,
			"proofGenTimeout": 30,
			"proofVerifyTime": 10,
			"recursionDepth": 3,
			"batchSize": 100,
		},
		"circuits": circuits,
		"trustedSetup": map[string]interface{}{
			"ceremonyDate": time.Now().Unix(),
			"participants": 100,
			"srsHash": "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			"verified": true,
		},
		"verifierRegistry": []map[string]interface{}{
			{
				"id": "plonk-verifier",
				"version": "1.0.0",
				"gasUsage": 300000,
			},
		},
		"privacyPools": []map[string]interface{}{
			{
				"id": "default-pool",
				"minDeposit": 1000000000,
				"maxDeposit": 1000000000000000,
				"participants": 0,
			},
		},
	}
	
	data, err := json.MarshalIndent(genesis, "", "  ")
	if err != nil {
		return err
	}
	
	genesisFile := filepath.Join(outputPath, "genesis.json")
	return os.WriteFile(genesisFile, data, 0644)
}

func generateCPUAffinityConfig(outputPath string) error {
	fmt.Println("📝 Generating CPU affinity configuration...")
	
	config := map[string]interface{}{
		"processorAffinity": map[string]interface{}{
			"enabled": true,
			"vmAssignments": map[string]int{
				"platformvm": 0,
				"evm": 1,
				"avm": 2,
				"aivm": 3,
				"bridgevm": 4,
				"mpcvm": 5,
				"quantumvm": 6,
				"zkvm": 7,
			},
			"loadBalancing": map[string]interface{}{
				"algorithm": "round-robin",
				"rebalanceInterval": 60,
				"cpuThreshold": 80,
			},
		},
		"gomaxprocs": 8,
		"threadPool": map[string]interface{}{
			"coreThreads": 8,
			"maxThreads": 16,
			"queueSize": 1000,
		},
	}
	
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	
	configFile := filepath.Join(outputPath, "cpu-affinity.json")
	return os.WriteFile(configFile, data, 0644)
}

func generateBootstrapConfig(outputPath, network string) error {
	fmt.Println("📝 Generating bootstrap configuration...")
	
	// Bootstrap nodes for different networks
	bootstrapNodes := map[string][]string{
		"mainnet": {
			"NodeID-A1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6@1.2.3.4:9651",
			"NodeID-B2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7@5.6.7.8:9651",
		},
		"testnet": {
			"NodeID-C3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8@9.10.11.12:9651",
		},
		"local": []string{},
	}
	
	config := map[string]interface{}{
		"network-id": network,
		"bootstrap-ips": "",
		"bootstrap-ids": "",
		"chain-aliases": map[string]string{
			"A": "aivm",
			"B": "bridgevm",
			"C": "evm",
			"M": "mpcvm",
			"P": "platform",
			"Q": "quantumvm",
			"X": "avm",
			"Z": "zkvm",
		},
	}
	
	if nodes, ok := bootstrapNodes[network]; ok && len(nodes) > 0 {
		config["bootstrap-ids"] = nodes[0] // Simplified for example
	}
	
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	
	configFile := filepath.Join(outputPath, "node-config.json")
	return os.WriteFile(configFile, data, 0644)
}

// Helper functions

func generateNodeIDSuffix() string {
	// In production, this would generate proper node IDs
	return "1111111111111111111111111111111111111111111"
}

func generateSubnetID(vmType string) string {
	// In production, this would generate proper subnet IDs
	suffixes := map[string]string{
		"aivm": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		"bridgevm": "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB",
		"mpcvm": "MMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMM",
		"quantumvm": "QQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQ",
		"zkvm": "ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ",
	}
	return suffixes[vmType]
}