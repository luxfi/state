package main

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/cockroachdb/pebble"
)

var _ = Describe("C-Chain Migration Smoke Tests", func() {
	var (
		projectRoot string
		luxdPath    string
		luxdPID     int
		rpcURL      = "http://localhost:9650/ext/bc/C/rpc"
	)

	BeforeSuite(func() {
		By("Setting up test environment")
		// Get project root
		wd, err := os.Getwd()
		Expect(err).NotTo(HaveOccurred())
		if filepath.Base(wd) == "test" {
			projectRoot = filepath.Dir(wd)
		} else {
			projectRoot = wd
		}

		// Check if luxd exists
		luxdPath = filepath.Join(projectRoot, "bin", "luxd")
		if _, err := os.Stat(luxdPath); os.IsNotExist(err) {
			By("Installing luxd from genesis branch")
			cmd := exec.Command("make", "deps")
			cmd.Dir = projectRoot
			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), string(output))
		}

		// Build genesis tool
		By("Building genesis tool")
		cmd := exec.Command("make", "build")
		cmd.Dir = projectRoot
		output, err := cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), string(output))
	})

	AfterSuite(func() {
		// Kill luxd if still running
		if luxdPID > 0 {
			exec.Command("kill", fmt.Sprintf("%d", luxdPID)).Run()
		}
	})

	Context("Full Migration Pipeline", func() {
		var expectedTip uint64

		It("should import subnet data to C-Chain format", func() {
			By("Running import process")
			cmd := exec.Command("make", "import")
			cmd.Dir = projectRoot
			output, err := cmd.CombinedOutput()
			
			fmt.Println("Import output:")
			fmt.Println(string(output))
			
			Expect(err).NotTo(HaveOccurred(), string(output))
			Expect(string(output)).To(ContainSubstring("Import complete!"))

			By("Reading tip height")
			tipBytes, err := os.ReadFile(filepath.Join(projectRoot, "runtime", "tip.txt"))
			Expect(err).NotTo(HaveOccurred())
			
			tipStr := strings.TrimSpace(string(tipBytes))
			expectedTip, err = strconv.ParseUint(tipStr, 10, 64)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("Expected tip height: %d\n", expectedTip)
		})

		It("should launch luxd with migrated data", func() {
			By("Starting luxd")
			cmd := exec.Command(luxdPath,
				"--db-dir", filepath.Join(projectRoot, "runtime"),
				"--network-id", "96369",
				"--staking-enabled=false",
				"--http-host", "0.0.0.0",
				"--http-port", "9650",
				"--chain-configs-dir", filepath.Join(projectRoot, "configs"),
				"--log-level", "info",
			)
			
			// Start luxd in background
			err := cmd.Start()
			Expect(err).NotTo(HaveOccurred())
			luxdPID = cmd.Process.Pid
			
			By("Waiting for RPC to be ready")
			Eventually(func() error {
				resp, err := http.Get(rpcURL)
				if err != nil {
					return err
				}
				resp.Body.Close()
				return nil
			}, 30*time.Second, 1*time.Second).Should(Succeed())
		})

		It("should return correct block height via RPC", func() {
			By("Calling eth_blockNumber")
			payload := `{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`
			resp, err := http.Post(rpcURL, "application/json", strings.NewReader(payload))
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			var result struct {
				Result string `json:"result"`
				Error  *struct {
					Message string `json:"message"`
				} `json:"error"`
			}
			
			err = json.NewDecoder(resp.Body).Decode(&result)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Error).To(BeNil(), "RPC error: %v", result.Error)

			// Convert hex to uint64
			height, err := strconv.ParseUint(strings.TrimPrefix(result.Result, "0x"), 16, 64)
			Expect(err).NotTo(HaveOccurred())
			
			fmt.Printf("Current block height: %d (0x%s)\n", height, result.Result)
			Expect(height).To(Equal(expectedTip), "Block height should match imported tip")
		})

		It("should return genesis block via RPC", func() {
			By("Calling eth_getBlockByNumber for block 0")
			payload := `{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["0x0",false],"id":1}`
			resp, err := http.Post(rpcURL, "application/json", strings.NewReader(payload))
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			var result struct {
				Result struct {
					Hash       string `json:"hash"`
					Number     string `json:"number"`
					ParentHash string `json:"parentHash"`
				} `json:"result"`
			}
			
			err = json.NewDecoder(resp.Body).Decode(&result)
			Expect(err).NotTo(HaveOccurred())
			
			fmt.Printf("Genesis block hash: %s\n", result.Result.Hash)
			Expect(result.Result.Hash).NotTo(BeEmpty())
			Expect(result.Result.Number).To(Equal("0x0"))
		})

		It("should show treasury balance > 1.9T LUX", func() {
			By("Checking treasury balance")
			treasury := "0x9011e888251ab053b7bd1cdb598db4f9ded94714"
			payload := fmt.Sprintf(`{"jsonrpc":"2.0","method":"eth_getBalance","params":["%s","latest"],"id":1}`, treasury)
			
			resp, err := http.Post(rpcURL, "application/json", strings.NewReader(payload))
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			var result struct {
				Result string `json:"result"`
			}
			
			err = json.NewDecoder(resp.Body).Decode(&result)
			Expect(err).NotTo(HaveOccurred())
			
			// Parse balance
			bal := new(big.Int)
			_, ok := bal.SetString(strings.TrimPrefix(result.Result, "0x"), 16)
			Expect(ok).To(BeTrue())

			// 1.9T with 18 decimals
			threshold := new(big.Int)
			threshold.SetString("1900000000000000000", 10)
			threshold.Mul(threshold, big.NewInt(1e12)) // Add 12 more zeros for trillion
			
			fmt.Printf("Treasury balance: %s wei (%s LUX)\n", 
				bal.String(), 
				new(big.Int).Div(bal, big.NewInt(1e18)).String())
			
			Expect(bal.Cmp(threshold)).To(BeNumerically(">=", 0), 
				"Treasury balance should be >= 1.9T LUX")
		})

		It("should respond to eth_getLogs", func() {
			By("Testing eth_getLogs with latest block")
			payload := `{"jsonrpc":"2.0","method":"eth_getLogs","params":[{"fromBlock":"latest","toBlock":"latest"}],"id":1}`
			
			resp, err := http.Post(rpcURL, "application/json", strings.NewReader(payload))
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			var result struct {
				Result []interface{} `json:"result"`
				Error  *struct {
					Message string `json:"message"`
				} `json:"error"`
			}
			
			err = json.NewDecoder(resp.Body).Decode(&result)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Error).To(BeNil())
			
			// Result should be an array (even if empty)
			Expect(result.Result).NotTo(BeNil())
			fmt.Printf("eth_getLogs returned %d logs\n", len(result.Result))
		})

		It("should have proper chain config", func() {
			By("Calling eth_chainId")
			payload := `{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}`
			
			resp, err := http.Post(rpcURL, "application/json", strings.NewReader(payload))
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			var result struct {
				Result string `json:"result"`
			}
			
			err = json.NewDecoder(resp.Body).Decode(&result)
			Expect(err).NotTo(HaveOccurred())
			
			chainID, err := strconv.ParseUint(strings.TrimPrefix(result.Result, "0x"), 16, 64)
			Expect(err).NotTo(HaveOccurred())
			
			fmt.Printf("Chain ID: %d\n", chainID)
			Expect(chainID).To(Equal(uint64(96369)), "Chain ID should be 96369")
		})
	})

	Context("Database Validation", func() {
		It("should have correct EVM database structure", func() {
			evmDB := filepath.Join(projectRoot, "runtime", "evm", "pebbledb")
			
			By("Opening EVM database")
			db, err := pebble.Open(evmDB, &pebble.Options{ReadOnly: true})
			Expect(err).NotTo(HaveOccurred())
			defer db.Close()

			By("Checking for key prefixes")
			prefixes := map[string][]byte{
				"headers":     []byte("evmh"),
				"bodies":      []byte("evmb"),
				"receipts":    []byte("evmr"),
				"canonical":   []byte("evmn"),
				"hashToNum":   []byte("evmH"),
			}

			for name, prefix := range prefixes {
				iter, _ := db.NewIter(&pebble.IterOptions{
					LowerBound: prefix,
					UpperBound: append(prefix, 0xff),
				})
				
				count := 0
				for iter.First(); iter.Valid() && count < 10; iter.Next() {
					count++
				}
				iter.Close()
				
				fmt.Printf("Found %d+ %s keys\n", count, name)
				Expect(count).To(BeNumerically(">", 0), 
					"Should have %s keys with prefix %x", name, prefix)
			}
		})
	})
})