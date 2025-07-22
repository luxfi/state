package commands

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/luxfi/genesis/pkg/bridge"
)

func NewScanNFTCommand() *cobra.Command {
	var (
		chain           string
		chainID         int64
		rpcURL          string
		contractAddress string
		projectName     string
		outputPath      string
		fromBlock       uint64
		toBlock         uint64
		batchSize       uint64
		includeMetadata bool
		crossReference  string
	)

	cmd := &cobra.Command{
		Use:   "scan-nft",
		Short: "Scan NFTs from external blockchain",
		Long: `Scan NFT collections from external blockchains like Ethereum or BSC.
This command identifies all NFT holders and their token IDs for genesis inclusion.`,
		Example: `  # Scan Lux Genesis NFTs from Ethereum
  teleport scan-nft \
    --chain ethereum \
    --contract 0x31e0f919c67cedd2bc3e294340dc900735810311 \
    --project lux \
    --output ./external/lux-nfts-ethereum.json

  # Scan any NFT collection for migration
  teleport scan-nft \
    --rpc https://polygon-rpc.com \
    --contract 0xYOUR_NFT_CONTRACT \
    --project my-project \
    --include-metadata

  # Scan from BSC and prepare for L2 deployment
  teleport scan-nft \
    --chain bsc \
    --contract 0xNFT_COLLECTION \
    --project custom-l2 \
    --cross-reference ./data/extracted/custom-200300 \
    --output ./external/custom-nfts-bsc.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate inputs
			if contractAddress == "" {
				return fmt.Errorf("contract address is required")
			}
			if projectName == "" {
				return fmt.Errorf("project name is required")
			}
			if chain == "" && rpcURL == "" {
				return fmt.Errorf("either --chain or --rpc must be specified")
			}

			config := bridge.NFTScannerConfig{
				Chain:           chain,
				ChainID:         chainID,
				RPCURL:          rpcURL,
				ContractAddress: contractAddress,
				ProjectName:     projectName,
				FromBlock:       fromBlock,
				ToBlock:         toBlock,
				BatchSize:       batchSize,
				IncludeMetadata: includeMetadata,
				CrossReference:  crossReference,
			}

			scanner, err := bridge.NewNFTScanner(config)
			if err != nil {
				return fmt.Errorf("failed to create scanner: %w", err)
			}

			log.Printf("Scanning NFTs from %s", contractAddress)
			log.Printf("Chain: %s", chain)
			log.Printf("Project: %s", projectName)

			// Run scan
			result, err := scanner.Scan()
			if err != nil {
				return fmt.Errorf("scan failed: %w", err)
			}

			// Display results
			fmt.Printf("\n✅ NFT scan completed!\n\n")
			fmt.Printf("Contract: %s\n", result.ContractAddress)
			fmt.Printf("Name: %s\n", result.CollectionName)
			fmt.Printf("Symbol: %s\n", result.Symbol)
			fmt.Printf("Total Supply: %d\n", result.TotalSupply)
			fmt.Printf("Unique Holders: %d\n", result.UniqueHolders)
			fmt.Printf("Blocks Scanned: %d to %d\n", result.FromBlock, result.ToBlock)

			// Show distribution
			if len(result.TypeDistribution) > 0 {
				fmt.Printf("\nNFT Type Distribution:\n")
				for nftType, count := range result.TypeDistribution {
					fmt.Printf("  %s: %d\n", nftType, count)
				}
			}

			// Show top holders
			if len(result.TopHolders) > 0 {
				fmt.Printf("\nTop 10 Holders:\n")
				for i, holder := range result.TopHolders {
					if i >= 10 {
						break
					}
					fmt.Printf("  %d. %s: %d NFTs\n", i+1, holder.Address, holder.Count)
				}
			}

			// Cross-reference results
			if crossReference != "" && result.CrossReferenceResult != nil {
				fmt.Printf("\nCross-Reference Results:\n")
				fmt.Printf("  Addresses on target chain: %d\n", result.CrossReferenceResult.FoundOnChain)
				fmt.Printf("  New addresses: %d\n", result.CrossReferenceResult.NewAddresses)
				fmt.Printf("  Missing from target: %d\n", result.CrossReferenceResult.MissingFromChain)
			}

			// Export results
			if outputPath == "" {
				outputPath = fmt.Sprintf("./nft-scan-%s-%s.json", projectName, chain)
			}

			if err := scanner.Export(outputPath); err != nil {
				return fmt.Errorf("failed to export results: %w", err)
			}

			fmt.Printf("\nResults exported to: %s\n", outputPath)
			fmt.Printf("Total NFTs found: %d\n", result.TotalNFTs)

			// Show staking information if applicable
			if result.StakingInfo != nil {
				fmt.Printf("\n⚡ Staking Configuration:\n")
				fmt.Printf("  Validator NFTs: %d (1M %s each)\n", 
					result.StakingInfo.ValidatorCount, projectName)
				fmt.Printf("  Total Staking Power: %s %s\n", 
					result.StakingInfo.TotalPower, projectName)
			}

			return nil
		},
	}

	// Flags
	cmd.Flags().StringVarP(&chain, "chain", "c", "", "Blockchain name (ethereum, bsc, polygon)")
	cmd.Flags().Int64Var(&chainID, "chain-id", 0, "Chain ID (auto-detected if not specified)")
	cmd.Flags().StringVar(&rpcURL, "rpc", "", "Custom RPC URL")
	cmd.Flags().StringVar(&contractAddress, "contract", "", "NFT contract address")
	cmd.Flags().StringVarP(&projectName, "project", "p", "", "Project name (lux, zoo, spc, hanzo)")
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output file path")
	cmd.Flags().Uint64Var(&fromBlock, "from-block", 0, "Start block (0 for earliest)")
	cmd.Flags().Uint64Var(&toBlock, "to-block", 0, "End block (0 for latest)")
	cmd.Flags().Uint64Var(&batchSize, "batch-size", 1000, "Block batch size for scanning")
	cmd.Flags().BoolVar(&includeMetadata, "include-metadata", false, "Fetch NFT metadata")
	cmd.Flags().StringVar(&crossReference, "cross-reference", "", "Cross-reference with extracted chain data")

	cmd.MarkFlagRequired("contract")
	cmd.MarkFlagRequired("project")

	return cmd
}