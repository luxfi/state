package commands

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/luxfi/genesis/pkg/bridge"
)

// NewScanEggHoldersCommand creates the scan-egg-holders command
func NewScanEggHoldersCommand() *cobra.Command {
	var (
		rpc        string
		fromBlock  uint64
		toBlock    uint64
		outputPath string
		format     string
	)

	cmd := &cobra.Command{
		Use:   "scan-egg-holders",
		Short: "Scan BSC for all EGG NFT holders",
		Long: `Scans the BSC blockchain to find all EGG NFT holders and their holdings.
		
The EGG NFT contract address on BSC is: 0x5bb68cf06289d54efde25155c88003be685356a8

This command will:
- Find all current EGG NFT holders
- Count how many EGGs each address holds
- Output results in a format matching the historical data`,
		Example: `  # Scan all EGG NFT holders
  teleport scan-egg-holders --output egg-holders.txt

  # Scan specific block range
  teleport scan-egg-holders --from-block 13000000 --to-block 14000000

  # Output as CSV
  teleport scan-egg-holders --format csv --output egg-holders.csv`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create NFT scanner config
			config := bridge.NFTScannerConfig{
				Chain:           "bsc",
				ChainID:         56,
				RPC:             rpc,
				ContractAddress: "0x5bb68cf06289d54efde25155c88003be685356a8", // EGG NFT
				ProjectName:     "egg",
				FromBlock:       fromBlock,
				ToBlock:         toBlock,
			}

			// Create scanner
			scanner, err := bridge.NewNFTScanner(config)
			if err != nil {
				return fmt.Errorf("failed to create scanner: %w", err)
			}
			defer scanner.Close()

			// Perform scan
			log.Printf("Scanning EGG NFT holders on BSC...")
			log.Printf("Contract: 0x5bb68cf06289d54efde25155c88003be685356a8")
			
			result, err := scanner.Scan()
			if err != nil {
				return fmt.Errorf("scan failed: %w", err)
			}

			// Get detailed NFT ownership data (optional)
			_, err = scanner.GetDetailedNFTs()
			if err != nil {
				log.Printf("Warning: could not get detailed NFT data: %v", err)
			}

			// Build holder map with counts
			holderMap := make(map[string]int)
			for _, nft := range result.NFTs {
				holderMap[strings.ToLower(nft.Owner)]++
			}

			// Sort holders by count (descending) then by address
			type holderInfo struct {
				address string
				count   int
			}
			holders := make([]holderInfo, 0, len(holderMap))
			for addr, count := range holderMap {
				holders = append(holders, holderInfo{addr, count})
			}
			sort.Slice(holders, func(i, j int) bool {
				if holders[i].count != holders[j].count {
					return holders[i].count > holders[j].count
				}
				return holders[i].address < holders[j].address
			})

			// Output results
			fmt.Printf("\nEGG NFT Holders Summary:\n")
			fmt.Printf("========================\n")
			fmt.Printf("Total NFTs: %d\n", result.TotalNFTs)
			fmt.Printf("Unique Holders: %d\n", len(holderMap))
			fmt.Printf("Collection: %s (%s)\n", result.CollectionName, result.Symbol)
			fmt.Printf("Block Scanned: %d\n", result.BlockScanned)
			fmt.Printf("\n")

			// Display holders in requested format
			if format == "csv" {
				fmt.Printf("Address,EggCount,ZooAmount,TokenName,TokenSymbol\n")
				for _, holder := range holders {
					// Calculate Zoo amount (each EGG = 4,200,000 ZOO)
					zooAmount := holder.count * 4200000
					fmt.Printf("%s,%d,%d,ZOO,ZOO\n", holder.address, holder.count, zooAmount)
				}
			} else {
				// Default format matching the provided data
				fmt.Printf("%-42s %15s %8s %12s %12s\n", "Address", "ZOO Amount", "Eggs", "TokenName", "TokenSymbol")
				fmt.Printf("%s\n", strings.Repeat("-", 90))
				
				totalEggs := 0
				for _, holder := range holders {
					// Calculate Zoo amount (each EGG = 4,200,000 ZOO)
					zooAmount := holder.count * 4200000
					fmt.Printf("%-42s %15d %8d %12s %12s\n", 
						holder.address, 
						zooAmount,
						holder.count,
						"ZOO",
						"ZOO")
					totalEggs += holder.count
				}
				
				fmt.Printf("%s\n", strings.Repeat("-", 90))
				fmt.Printf("%-42s %15s %8d\n", "TOTAL", "", totalEggs)
			}

			// Save to file if specified
			if outputPath != "" {
				// TODO: Implement file output
				log.Printf("\nResults would be saved to: %s", outputPath)
			}

			// Show comparison with expected holders if available
			fmt.Printf("\n\nKnown EGG Holders from Historical Data:\n")
			fmt.Printf("=======================================\n")
			knownHolders := getKnownEggHolders()
			matchCount := 0
			
			for addr, expectedCount := range knownHolders {
				actualCount := holderMap[strings.ToLower(addr)]
				status := "❌"
				if actualCount == expectedCount {
					status = "✅"
					matchCount++
				}
				fmt.Printf("%s %s: Expected %d, Found %d\n", status, addr, expectedCount, actualCount)
			}
			
			fmt.Printf("\nMatched: %d/%d known holders\n", matchCount, len(knownHolders))

			return nil
		},
	}

	// Add flags
	cmd.Flags().StringVar(&rpc, "rpc", "", "BSC RPC endpoint (default: BSC public RPC)")
	cmd.Flags().Uint64Var(&fromBlock, "from-block", 0, "Start block for scanning")
	cmd.Flags().Uint64Var(&toBlock, "to-block", 0, "End block for scanning (default: latest)")
	cmd.Flags().StringVar(&outputPath, "output", "", "Output file path")
	cmd.Flags().StringVar(&format, "format", "text", "Output format (text, csv)")

	return cmd
}

// getKnownEggHolders returns the known EGG holders from the provided data
func getKnownEggHolders() map[string]int {
	return map[string]int{
		"0x9bfb8a065a884c56e216d93a72f6fb88f3919b55": 20, // 15 + 2 + 3
		"0xaa64006a3a14e16d58933c1fad7ff4f1468c5efd": 20,
		"0x1db1f644c2c0bca40473ce48260a1ff312449618": 6,
		"0xfe128356d8b085c807c4604f750bb236dbdb6082": 20,
		"0xfdd17d7fbe677516ffe7ffa3e1d200b559d79fe5": 20,
		"0xdecc54d2e97f44778143752254a05c1152d39aa7": 16, // 6 + 10
		"0xfe457c54857b39dac766844803e2fd90c92600c4": 5,
		"0xeefa6898191c92690bd8ef1bb972096d2971bcf0": 6,
		"0xb58bab9ea70256195ba46243787b5685475ad137": 120, // 20 + 100
		"0x826c9f14060c027c321e3fb0b4721db042cc1d0d": 20, // 10 + 10
		"0xf831b395cb0e1a014257ae6baa79fcd9efa538f2": 8, // 4 + 1 + 3
		"0x05c502113c898c6885faf6919f7054d66f563f9e": 20,
		"0xaf82c2f30ecf48d20e55652338553f549b585d34": 24, // 20 + 4
		"0xabf62700027399c659c8afe0a9fb12a446b504ef": 20,
		"0xb23e3baa84017ed6efd15adf978858b6a8f39435": 6, // 3 + 1 + 1 + 1 + 3
		"0xc6c231f34637145ee4fd2174309e465ef56ba7b0": 20,
		"0xa68a21931ceff6d29070923d3e9c7abac710dcf9": 20,
		"0x12cccea8152c36e0029d72a202a5f48e166618d7": 20,
		"0x2e5489309ba8ac06ccde965a9f190cdb37e0a739": 20,
		"0x340a5153500db300795c46b327d4ce476d7f8950": 20,
		"0xdd50a3b91025a44ead57dcb109b7b5a30a28f80f": 20,
		"0xaca3d99bcedb995143d4ac3e2acf7c099b5fe71d": 20,
		"0x6a6863b14413910b6a9054f6face292e031ec6da": 20,
		"0x23ffa003d2d6ca396708cb41d482907fbd19b0e1": 3,
		"0xfee726941989d44c5ce6ea113cc56b08ea2b3e41": 20,
		"0x87b4983e6708bc4ef0ec6b260da53aa6ca74c31f": 7,
		"0x9b2f2d070f842b690474ed0ec3ea37de545a601f": 16, // 2 + 2 + 4 + 2 + 4 + 2
		"0xfaf765fff46bc3edfc4ab3214a8aea127ebb9c66": 10, // 4 + 6
		"0xb2a058d62d4674c5c0d17fa5f318a85c4738d1fb": 10, // 3 + 4 + 3
		"0x7094c837d3a578090caeb9be4538773d113ee0ee": 10,
		"0xbf6acf74d710f01b2e34cdf37b8e67f2faa4f86d": 3,
		"0x3333643cd0e5bd81dd2007e8fd0abdfdc092002e": 4, // 2 + 2
		"0x21180db4a88a0aa5f0573d77a1bfcf7d7ea4e3ef": 6, // 2 + 3 + 1 + 2
		"0x6a47b6a5cca9d3c087e9578a0a3a6afe5b5c4479": 20, // 4 + 4 + 12
		"0xd444d13a8b3975b17baa7ef27e6990352bf34eb5": 20,
		"0x5a3d8854575fb1a93c3d7f3009156390a3ed76ad": 10, // 5 + 2 + 3
		"0x2dfdaf97d96e9a39205c6249070e6e2820e71952": 8, // 3 + 3 + 1 + 1 + 2
		"0x7a04a145ca905f19cea9cde5e5ab052732a13c76": 20, // 4 + 16
		"0x54afc4f31610a381ee0b812ed54936d4e5320935": 3,
		"0xfd83f112c966374fec0bedacc21ae2d45a3f39fb": 8,
		"0x4cf74728230bbab99bb3c4899f6a40e0e10930c9": 1,
		"0xf997b8999c827606e431e8db4022040ba31c1b6f": 7, // 1 + 6
		"0x225ee8a7d1b460f42f6e29ae5d7abd33d05fac8e": 15, // 6 + 5 + 4
		"0x7dbf183fbe9128fb5f56c214fef04b2cebef8a01": 1,
		"0x8778c1c607774e92fa63d20fc3321dbb7c9f8efa": 20, // 10 + 10
		"0xb32ddfdfad90c57e475916ef014d8bec3d0e3395": 6, // 2 + 1 + 3
		"0xb85d8c7a0598b7248f480661bc285f30092825cf": 12, // 2 + 10
		"0x4b83911c955a007c781eb60d95d959b272d6dc10": 4,
		"0x4c9c2d155961a75876533e9151353fa1290d3168": 2,
		"0xd9b4db1173b2311cecc1a6d24119ca4735405bde": 20,
		"0x78876ed849c98c817120d08e94184852e0e5340d": 7, // 5 + 2
		"0x7d0cd6a6c68a46b037208c0a289bf34122d31672": 20, // 4 + 11 + 5
		"0xa99b3e8dacace358265cd78aedeb891ccf0ca319": 12, // 4 + 4 + 4
		"0xc62ed5ddf2c97538c4268615e46b7a9991cd385f": 5,
		"0xc3adc59b29e40fdf6fafdc700898d401b4e64e97": 4,
		"0x0ba9bd3225665f397820651e172cfb5e43d8991e": 2, // 1 + 1
		"0x3556cc9a006b384d854a9fc688c891a351d3c144": 8, // 4 + 1 + 3 + 6
		"0x1088de1ac37e26a16814c0d1d20ef25350490879": 20,
		"0xd08d35f5c862971a7ee8a5d1d66e694c8f260ca9": 4,
		"0xc526fdc9950cfa85e77309c5a3fa64b3f34c6260": 20, // 5 + 5 + 10
		"0x048d84298d42413020b4f75dc8a2efc9e003630a": 1,
		"0xe1446c602565881677f0b9cf9cafea5960af6e4d": 4, // 2 + 2
		"0x4995a137ada6447fe586d7192f81443ce76d3e05": 1,
		"0xf2216e820192c019405ec30f6e93ae9401031855": 1,
		"0xfef62a7b791cd9dfc137ca2bd56a55abbb7d1738": 15, // 2 + 3 + 10
		"0xbb8008b60dc3b6a7ac2957a2035be5145a1bde60": 10, // 1 + 9
		"0xba0ba1f733f347634ff4a253bb38b6652d866ea7": 6,
		"0x5926a00a9cc4d2573a11f064c787f097f6b4d209": 16, // 2 + 8 + 6
		"0x712fc6e43bfd2e9a1292343c96a63a5d7862a350": 2,
		"0x9f5fef290aba8ba70a9d4439ef73ce337cfe2b6e": 14, // 1 + 1 + 2 + 8 + 2
		"0x6cde195e0da3b0d8cb9611b08bc0d798fccd999b": 7, // 1 + 5 + 1
		"0xadebf3db63610f6a07cce88f14bf58de1c45e264": 10,
		"0xb01763c5eef94d9b1ebea4af9184160ab184e8ce": 3, // 1 + 2
		"0xf242ee88439e4d9a70ad336d1fc84a59d9e794fd": 20, // 7 + 5 + 8
		"0xc9c6de34a46bd90264a36c791a6d751ef2455a83": 6,
		"0xf1a4d066cbefc59f3335fdcc6cf35ea62874e263": 17, // 10 + 7
		"0x008f01f3f5d15abecc72247d140bfd0226f6a283": 2, // 1 + 1
		"0xd858299ca7be4087c942ecf8dd52df2aa81c7764": 1,
		"0xab67be0ef93acafb0ee9760f719b6e31573b5969": 3, // 1 + 2
		"0x6d068a72c8bc06caf7b21e1cd2e552bd3e2fe58b": 20, // 15 + 5
		"0xbd1a685bc7b3a2a68fbc611d6dcff02cfa1706ef": 20, // 7 + 13
		"0x8a755f40c448666e5a8651adac598d13897a8a88": 2,
		"0x00e53da366eeeb025b6847affd231c611fb38c36": 1,
		"0x03a0f70fae2c6da03477b395521e2b6fe9b18dd6": 20,
		"0xb8fabdd6872eb32d70bf2b95fccefbf88fdb2291": 10,
		"0x470642b893428b60b7f2dfdd2109fc24ec00cd8a": 1,
		"0xeb26fa4adf849d0f18de2d811e5f810bfd8c8b74": 20,
		"0x52dbe045740d5ccb6c6a60ccef379c85eea91f70": 20,
		"0x4c1b74395e51292c87bc5c14f582125158c370c1": 1,
		"0xd881f508b46b2537c854b17deb413d7195bd5859": 20,
		"0x5e9c755ca385e793cdc801f374cfd1aa64b62b39": 1,
		"0xdbddb0eefc91881cdd4139821d3262fab45f1a31": 4,
		"0x66f3b12167edd9347c58": 2,
		"0xc1b746a30bc69e72efa1bedd981aca6caa34ace2": 10,
		"0xa8e216efa99fab75120f07a021779f5437a4601e": 10,
		"0x9882fcf8dcf9bbef9306bc5294911964a0320137": 30, // 20 + 10
		"0x201376acc2380ad2702955b550ebade9977d0866": 10,
		"0xb82da2db4b24d3be64020ac98963c9769893ea6d": 10,
		"0x1495809b84f07cb1364560c3f5700486c6f92cf1": 20,
		"0x5ea3c17a87014bd1df7a817ac0c4139525543ab5": 20,
		// Special addresses from the data
		"0x4db30e41ad76ecc5533d46bd68493648ca0400ae": 5,
		"0xffdb31285961d44d40c404566e9de9080b1abd50": 2,  // otc
		"0xc06c7c6ec618de992d597d8e347669ea44ede2bc": 2,  // jules
		"0x6762ff916de1b315da56f4fa7b78f39aa60d9f4c": 1,  // sean
		"0x28dad8427f127664365109c4a9406c8bc7844718": 1300, // remaining eggs
		"0x95a7b934860942e903c47d85041e263ea9167de8": 160, // zach
	}
}