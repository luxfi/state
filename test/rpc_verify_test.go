package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("luxd smoke-test (subnet-96369)", Serial, func() {
	var (
		repoRoot   string
		scriptPath string
		pidFile    string
		rpcURL     string
	)

	BeforeEach(func() {
		var err error
		repoRoot, err = filepath.Abs("../")
		Expect(err).NotTo(HaveOccurred())

		scriptPath = filepath.Join(repoRoot, "scripts", "migrate-and-run-96369.sh")
		pidFile = filepath.Join(repoRoot, "runtime", "lux-96369-migrated", "luxd.pid")
		rpcURL = "http://127.0.0.1:9630/ext/bc/C/rpc"
	})

	AfterEach(func() {
		// Clean up: kill the node if it's still running
		if pid, err := ioutil.ReadFile(pidFile); err == nil {
			exec.Command("kill", string(bytes.TrimSpace(pid))).Run()
			time.Sleep(2 * time.Second)
		}
	})

	It("boots and serves correct height", func() {
		By("Running migration and starting luxd")
		cmd := exec.Command(scriptPath)
		cmd.Env = os.Environ()
		output, err := cmd.CombinedOutput()

		// Print output for debugging
		fmt.Printf("Script output:\n%s\n", string(output))

		Expect(err).ShouldNot(HaveOccurred(),
			"migrate-and-run-96369.sh failed: %s", string(output))

		// Check that key markers are in output
		Expect(string(output)).To(ContainSubstring("RPC ready"))
		Expect(string(output)).To(ContainSubstring("luxd for subnet-96369 is live"))

		By("Verifying RPC is accessible")
		// Give it a moment to stabilize
		time.Sleep(2 * time.Second)

		// Test eth_blockNumber
		blockNumResp := rpcCall(rpcURL, "eth_blockNumber", []interface{}{})
		Expect(blockNumResp).NotTo(BeNil())

		blockNumHex, ok := blockNumResp.(string)
		Expect(ok).To(BeTrue(), "eth_blockNumber should return a string")
		Expect(blockNumHex).To(HavePrefix("0x"))

		// Convert to decimal
		blockNum := new(big.Int)
		blockNum.SetString(blockNumHex[2:], 16)
		fmt.Printf("Current block number: %s\n", blockNum.String())

		By("Verifying treasury balance")
		treasury := "0x9011e888251ab053b7bd1cdb598db4f9ded94714"
		balanceResp := rpcCall(rpcURL, "eth_getBalance", []interface{}{treasury, "latest"})
		Expect(balanceResp).NotTo(BeNil())

		balanceHex, ok := balanceResp.(string)
		Expect(ok).To(BeTrue(), "eth_getBalance should return a string")

		// Convert to big.Int
		balance := new(big.Int)
		balance.SetString(balanceHex[2:], 16)

		// Check if balance is > 1.9T LUX (1.9 * 10^30 wei)
		minBalance := new(big.Int)
		minBalance.SetString("1900000000000000000000000000000", 10)

		Expect(balance.Cmp(minBalance)).To(BeNumerically(">=", 0),
			"Treasury balance %s should be >= %s", balance.String(), minBalance.String())

		By("Verifying chain ID")
		chainIDResp := rpcCall(rpcURL, "eth_chainId", []interface{}{})
		Expect(chainIDResp).NotTo(BeNil())

		chainIDHex, ok := chainIDResp.(string)
		Expect(ok).To(BeTrue(), "eth_chainId should return a string")
		Expect(chainIDHex).To(Equal("0x17971"), "Chain ID should be 96369")
	})

	It("can retrieve historical blocks", func() {
		Skip("Requires node to be already running")

		By("Getting block 0 (genesis)")
		block0 := rpcCall(rpcURL, "eth_getBlockByNumber", []interface{}{"0x0", false})
		Expect(block0).NotTo(BeNil())

		blockMap, ok := block0.(map[string]interface{})
		Expect(ok).To(BeTrue(), "Block should be a map")
		Expect(blockMap["number"]).To(Equal("0x0"))

		By("Getting a recent block")
		latestBlock := rpcCall(rpcURL, "eth_getBlockByNumber", []interface{}{"latest", false})
		Expect(latestBlock).NotTo(BeNil())

		latestMap, ok := latestBlock.(map[string]interface{})
		Expect(ok).To(BeTrue(), "Latest block should be a map")
		Expect(latestMap["hash"]).NotTo(BeEmpty())
	})
})

// Helper function to make RPC calls
func rpcCall(url string, method string, params []interface{}) interface{} {
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
		"id":      1,
	}

	jsonData, err := json.Marshal(payload)
	Expect(err).NotTo(HaveOccurred())

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	Expect(err).NotTo(HaveOccurred())
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	Expect(err).NotTo(HaveOccurred())

	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	Expect(err).NotTo(HaveOccurred())

	if errObj, ok := result["error"]; ok {
		Fail(fmt.Sprintf("RPC error: %v", errObj))
	}

	return result["result"]
}
