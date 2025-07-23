package integration_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Network Integration", Ordered, func() {
	var (
		luxdPath     string
		cliPath      string
		baseDir      string
		genesisDir   string
	)

	BeforeAll(func() {
		// Set up paths
		pwd, err := os.Getwd()
		Expect(err).NotTo(HaveOccurred())
		
		// Find genesis root (might be running from tests/ subdirectory)
		genesisDir = pwd
		for !strings.HasSuffix(genesisDir, "genesis") && genesisDir != "/" {
			genesisDir = filepath.Dir(genesisDir)
		}
		
		luxdPath = filepath.Join(genesisDir, "bin/luxd")
		cliPath = filepath.Join(genesisDir, "bin/lux")
		baseDir = filepath.Join(os.TempDir(), "lux-test-"+time.Now().Format("20060102-150405"))
		
		// Verify binaries exist
		Expect(luxdPath).To(BeAnExistingFile(), "luxd binary not found")
		Expect(cliPath).To(BeAnExistingFile(), "lux-cli binary not found")
		
		// Create test directory
		Expect(os.MkdirAll(baseDir, 0755)).To(Succeed())
		
		DeferCleanup(func() {
			os.RemoveAll(baseDir)
		})
	})

	Describe("5-Node Primary Network", func() {
		var networkDir string
		var session *gexec.Session

		BeforeEach(func() {
			networkDir = filepath.Join(baseDir, "5-node-network")
			Expect(os.MkdirAll(networkDir, 0755)).To(Succeed())
		})

		AfterEach(func() {
			if session != nil {
				session.Terminate().Wait()
			}
		})

		It("should start a 5-node local network", func() {
			By("Creating network configuration")
			cmd := exec.Command(cliPath, "network", "create",
				"--network-name", "test-network",
				"--node-count", "5",
				"--output-dir", networkDir,
			)
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))

			By("Starting the network")
			cmd = exec.Command(cliPath, "network", "start",
				"--network-dir", networkDir,
			)
			session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			
			By("Waiting for network to be healthy")
			Eventually(func() bool {
				healthCmd := exec.Command(cliPath, "network", "health",
					"--network-dir", networkDir,
				)
				return healthCmd.Run() == nil
			}, 2*time.Minute, 5*time.Second).Should(BeTrue())

			By("Verifying all 5 nodes are running")
			statusCmd := exec.Command(cliPath, "network", "status",
				"--network-dir", networkDir,
			)
			output, err := statusCmd.Output()
			Expect(err).NotTo(HaveOccurred())
			Expect(string(output)).To(ContainSubstring("5 nodes running"))
		})
	})

	Describe("C-Chain Data Import", func() {
		var (
			chainDataPath string
			pebbleDBPath  string
		)

		BeforeAll(func() {
			chainDataPath = filepath.Join(genesisDir, "chaindata/lux-96369")
			pebbleDBPath = filepath.Join(genesisDir, "pebbledb/lux-96369")
		})

		It("should convert LevelDB to PebbleDB for 96369", func() {
			Skip("Requires chaindata to be present")
			
			By("Checking if chaindata exists")
			Expect(chainDataPath).To(BeADirectory())

			By("Running conversion")
			cmd := exec.Command("go", "run",
				filepath.Join(genesisDir, "scripts/convert.go"),
				"-chain", "96369",
				"-input", chainDataPath,
				"-output", pebbleDBPath,
			)
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, 5*time.Minute).Should(gexec.Exit(0))

			By("Verifying PebbleDB was created")
			Expect(pebbleDBPath).To(BeADirectory())
		})

		It("should export 7777 accounts for X-Chain funding", func() {
			Skip("Requires 7777 chaindata")
			
			By("Exporting 7777 accounts to CSV")
			cmd := exec.Command("go", "run",
				filepath.Join(genesisDir, "scripts/export_7777_accounts.go"),
			"--db-path", filepath.Join(genesisDir, "chaindata/lux-genesis-7777/db"),
				"--output", filepath.Join(baseDir, "7777-accounts.csv"),
				"--exclude-treasury", "0x9011E888251AB053B7bD1cdB598Db4f9DEd94714",
			)
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, 2*time.Minute).Should(gexec.Exit(0))

			By("Verifying CSV was created")
			csvPath := filepath.Join(baseDir, "7777-accounts.csv")
			Expect(csvPath).To(BeAnExistingFile())
		})

		It("should import 96369 C-Chain data into running network", func() {
			Skip("Requires running network and converted data")
			
			By("Importing 96369 data to C-Chain")
			cmd := exec.Command(cliPath, "blockchain", "import", "c-chain",
				"--genesis-file", filepath.Join(genesisDir, "configs/genesis-96369.json"),
				"--db-path", pebbleDBPath,
				"--network-id", "96369",
			)
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, 10*time.Minute).Should(gexec.Exit(0))

			By("Verifying import success")
			// Check RPC endpoint
			Eventually(func() error {
				cmd := exec.Command("curl", "-X", "POST",
					"-H", "Content-Type: application/json",
					"-d", `{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`,
					"http://localhost:9630/ext/bc/C/rpc",
				)
				_, err := cmd.Output()
				return err
			}, 2*time.Minute, 5*time.Second).Should(Succeed())
		})
	})

	Describe("L2 Subnet Deployment", func() {
		It("should deploy ZOO as L2 subnet", func() {
			Skip("Requires running primary network")
			
			By("Creating ZOO subnet")
			cmd := exec.Command(cliPath, "subnet", "create",
				"--subnet-name", "zoo",
				"--chain-id", "200200",
				"--genesis-file", filepath.Join(genesisDir, "chaindata/configs/zoo-mainnet-200200/genesis.json"),
			)
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))
			
			// Extract subnet ID from output
			output := string(session.Out.Contents())
			fmt.Fprintf(GinkgoWriter, "Subnet creation output: %s\n", output)
			// Parse subnet ID from output
		})

		It("should deploy SPC as L2 subnet", func() {
			Skip("Requires running primary network")
			
			By("Creating SPC subnet")
			cmd := exec.Command(cliPath, "subnet", "create",
				"--subnet-name", "spc",
				"--chain-id", "36911",
				"--genesis-file", filepath.Join(genesisDir, "chaindata/configs/spc-mainnet-36911/genesis.json"),
			)
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))
		})

		It("should deploy Hanzo as L2 subnet", func() {
			Skip("Requires running primary network")
			
			By("Creating Hanzo subnet")
			cmd := exec.Command(cliPath, "subnet", "create",
				"--subnet-name", "hanzo",
				"--chain-id", "36963",
				"--genesis-file", filepath.Join(genesisDir, "chaindata/configs/hanzo-mainnet-36963/genesis.json"),
			)
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))
		})
	})

	Describe("7777 Dev Mode", func() {
		var devSession *gexec.Session

		AfterEach(func() {
			if devSession != nil {
				devSession.Terminate().Wait()
			}
		})

		It("should run 7777 chain in dev mode", func() {
			By("Converting 7777 data if needed")
    pebbleDBPath := filepath.Join(genesisDir, "pebbledb/lux-genesis-7777")
			if _, err := os.Stat(pebbleDBPath); os.IsNotExist(err) {
				cmd := exec.Command("go", "run",
					filepath.Join(genesisDir, "scripts/convert.go"),
					"-chain", "7777",
				)
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session, 5*time.Minute).Should(gexec.Exit(0))
			}

			By("Starting luxd in dev mode with 7777 data")
			cmd := exec.Command(luxdPath,
				"--dev",
				"--network-id=7777",
				"--db-dir", pebbleDBPath,
				"--chain-config-dir", filepath.Join(genesisDir, "configs/lux-genesis-7777"),
				"--http-port=9630",
				"--staking-port=9631",
			)
			
			var err error
			devSession, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for node to be ready")
			Eventually(func() error {
				cmd := exec.Command("curl", "-s",
					"http://localhost:9630/ext/health",
				)
				_, err := cmd.Output()
				return err
			}, 2*time.Minute, 5*time.Second).Should(Succeed())

			By("Verifying 7777 chain is accessible")
			cmd = exec.Command("curl", "-X", "POST",
				"-H", "Content-Type: application/json",
				"-d", `{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}`,
				"http://localhost:9630/ext/bc/C/rpc",
			)
			output, err := cmd.Output()
			Expect(err).NotTo(HaveOccurred())
			Expect(string(output)).To(ContainSubstring("0x1e61")) // 7777 in hex
		})
	})
})
