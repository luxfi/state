package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Helper to get the genesis binary path
func genesisBin() string {
	root, _ := filepath.Abs("../")
	return filepath.Join(root, "bin", "genesis")
}

// Helper to run genesis command
func genesis(args ...string) (string, error) {
	return run(genesisBin(), args...)
}

// Helper to run genesis command (must succeed)
func mustGenesis(args ...string) string {
	return mustRun(genesisBin(), args...)
}

// Helper to get absolute path to source file
func srcFile(name string) string {
	root, _ := filepath.Abs("../")
	return filepath.Join(root, name)
}

// Helper to run a command and return output
func mustRun(name string, args ...string) string {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		panic(fmt.Sprintf("Command failed: %s %v\nOutput: %s\nError: %v", 
			name, args, string(output), err))
	}
	return strings.TrimSpace(string(output))
}

// Helper to run a command and return output and error
func run(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// Wait for RPC port to be available
func waitRPC(port string) error {
	for i := 0; i < 20; i++ {
		conn, err := net.DialTimeout("tcp", "127.0.0.1:"+port, 300*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("RPC port %s never came up", port)
}

// Build genesis tool if requested
func buildGenesisTool() error {
	if os.Getenv("GINKGO_BUILD_TOOLING") != "true" {
		// Check if binary already exists
		if _, err := os.Stat(genesisBin()); err == nil {
			return nil
		}
	}
	
	root, _ := filepath.Abs("../")
	cmd := exec.Command("go", "build", 
		"-o", genesisBin(),
		"./cmd/genesis")
	cmd.Dir = root
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("Failed to build genesis tool: %v\nOutput: %s", 
			err, string(output))
	}
	return nil
}