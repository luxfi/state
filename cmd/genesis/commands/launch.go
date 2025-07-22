package commands

import (
	"fmt"
	"log"
	"time"

	"github.com/spf13/cobra"
	"github.com/luxfi/genesis/pkg/genesis"
)

func NewLaunchCommand() *cobra.Command {
	var (
		networkName    string
		genesisPath    string
		dataDir        string
		validators     int
		devMode        bool
		rpcPort        int
		p2pPort        int
		enableAPIs     []string
		logLevel       string
		detached       bool
		automining     bool
	)

	cmd := &cobra.Command{
		Use:   "launch",
		Short: "Launch a network with genesis file",
		Long: `Launch a Lux Network node with the specified genesis configuration.
This command starts a fully configured node with the given genesis state.`,
		Example: `  # Launch LUX mainnet node
  genesis launch \
    --network lux-mainnet \
    --genesis ./genesis/lux-mainnet-96369.json \
    --validators 5

  # Launch development node with automining
  genesis launch \
    --network lux-dev \
    --genesis ./genesis/lux-dev.json \
    --dev-mode \
    --automining \
    --rpc-port 9650

  # Launch detached production node
  genesis launch \
    --network zoo-mainnet \
    --genesis ./genesis/zoo-mainnet.json \
    --data-dir /data/zoo \
    --detached \
    --log-level info`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if networkName == "" {
				return fmt.Errorf("network name is required")
			}
			if genesisPath == "" {
				return fmt.Errorf("genesis file path is required")
			}

			config := genesis.LauncherConfig{
				NetworkName: networkName,
				GenesisPath: genesisPath,
				DataDir:     dataDir,
				Validators:  validators,
				DevMode:     devMode,
				RPCPort:     rpcPort,
				P2PPort:     p2pPort,
				EnableAPIs:  enableAPIs,
				LogLevel:    logLevel,
				Detached:    detached,
				Automining:  automining,
			}

			launcher, err := genesis.NewLauncher(config)
			if err != nil {
				return fmt.Errorf("failed to create launcher: %w", err)
			}

			// Validate genesis before launch
			log.Printf("Validating genesis file: %s", genesisPath)
			if err := launcher.ValidateGenesis(); err != nil {
				return fmt.Errorf("genesis validation failed: %w", err)
			}
			log.Printf("✅ Genesis validation passed")

			// Prepare launch
			log.Printf("Preparing to launch %s network", networkName)
			if devMode {
				log.Printf("🔧 Development mode enabled")
				if automining {
					log.Printf("⛏️  Automining enabled")
				}
			}

			// Show configuration
			fmt.Printf("\nNetwork Configuration:\n")
			fmt.Printf("======================\n")
			fmt.Printf("Network: %s\n", networkName)
			fmt.Printf("Genesis: %s\n", genesisPath)
			fmt.Printf("Data Directory: %s\n", config.DataDir)
			fmt.Printf("RPC Port: %d\n", rpcPort)
			fmt.Printf("P2P Port: %d\n", p2pPort)
			fmt.Printf("Validators: %d\n", validators)
			fmt.Printf("APIs Enabled: %v\n", enableAPIs)
			fmt.Printf("Log Level: %s\n", logLevel)

			if !detached {
				fmt.Printf("\n⚡ Starting node (press Ctrl+C to stop)...\n\n")
			}

			// Launch the network
			result, err := launcher.Launch()
			if err != nil {
				return fmt.Errorf("launch failed: %w", err)
			}

			if detached {
				fmt.Printf("\n✅ Network launched in background!\n")
				fmt.Printf("Process ID: %d\n", result.ProcessID)
				fmt.Printf("Log file: %s\n", result.LogFile)
			} else {
				// Show real-time information
				fmt.Printf("\n🚀 Network is running!\n\n")
				fmt.Printf("RPC Endpoint: %s\n", result.RPCEndpoint)
				fmt.Printf("WebSocket: %s\n", result.WSEndpoint)
				fmt.Printf("Metrics: %s\n", result.MetricsEndpoint)
				
				if result.ExplorerURL != "" {
					fmt.Printf("Explorer: %s\n", result.ExplorerURL)
				}

				fmt.Printf("\nNode ID: %s\n", result.NodeID)
				fmt.Printf("Network ID: %d\n", result.NetworkID)
				fmt.Printf("Chain ID: %d\n", result.ChainID)

				// Monitor node status
				fmt.Printf("\n📊 Node Status:\n")
				fmt.Printf("================\n")

				// Keep running and show periodic status
				ticker := time.NewTicker(10 * time.Second)
				defer ticker.Stop()

				for {
					select {
					case <-ticker.C:
						status, err := launcher.GetStatus()
						if err != nil {
							log.Printf("Failed to get status: %v", err)
							continue
						}

						fmt.Printf("\r⏱️  Uptime: %s | 📦 Height: %d | 👥 Peers: %d | 💾 DB Size: %s",
							status.Uptime, status.BlockHeight, status.PeerCount, status.DatabaseSize)
					case <-cmd.Context().Done():
						fmt.Printf("\n\n⏹️  Stopping node...\n")
						if err := launcher.Stop(); err != nil {
							return fmt.Errorf("failed to stop node: %w", err)
						}
						fmt.Printf("✅ Node stopped successfully\n")
						return nil
					}
				}
			}

			return nil
		},
	}

	// Flags
	cmd.Flags().StringVarP(&networkName, "network", "n", "", "Network name")
	cmd.Flags().StringVarP(&genesisPath, "genesis", "g", "", "Path to genesis file")
	cmd.Flags().StringVarP(&dataDir, "data-dir", "d", "", "Data directory (default: ~/.lux/<network>)")
	cmd.Flags().IntVar(&validators, "validators", 1, "Number of validators (dev mode)")
	cmd.Flags().BoolVar(&devMode, "dev-mode", false, "Enable development mode")
	cmd.Flags().IntVar(&rpcPort, "rpc-port", 9650, "RPC port")
	cmd.Flags().IntVar(&p2pPort, "p2p-port", 9651, "P2P port")
	cmd.Flags().StringSliceVar(&enableAPIs, "enable-apis", []string{"eth", "web3", "net"}, "APIs to enable")
	cmd.Flags().StringVar(&logLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	cmd.Flags().BoolVar(&detached, "detached", false, "Run in background")
	cmd.Flags().BoolVar(&automining, "automining", false, "Enable automining (dev mode only)")

	cmd.MarkFlagRequired("network")
	cmd.MarkFlagRequired("genesis")

	return cmd
}