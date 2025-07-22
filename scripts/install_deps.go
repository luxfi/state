package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	// Get binary directory
	binDir := filepath.Join(".", "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		log.Fatalf("Failed to create bin directory: %v", err)
	}

	fmt.Println("Installing LUX dependencies from GitHub...")

	// Install luxd
	fmt.Println("Installing luxd...")
	cmd := exec.Command("go", "install", "-v", "github.com/luxfi/node/cmd/luxd@latest")
	cmd.Env = append(os.Environ(), "GOBIN="+absPath(binDir))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("Failed to install luxd: %v", err)
	}

	// Install lux-cli as 'lux'
	fmt.Println("Installing lux-cli...")
	cmd = exec.Command("go", "install", "-v", "github.com/luxfi/cli/cmd/lux-cli@latest")
	cmd.Env = append(os.Environ(), "GOBIN="+absPath(binDir))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("Failed to install lux-cli: %v", err)
	}

	// Rename lux-cli to lux
	oldPath := filepath.Join(binDir, "lux-cli")
	newPath := filepath.Join(binDir, "lux")
	if err := os.Rename(oldPath, newPath); err != nil {
		log.Printf("Warning: Failed to rename lux-cli to lux: %v", err)
	}

	// Install ginkgo for testing
	fmt.Println("Installing ginkgo...")
	cmd = exec.Command("go", "install", "-v", "github.com/onsi/ginkgo/v2/ginkgo@latest")
	cmd.Env = append(os.Environ(), "GOBIN="+absPath(binDir))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("Failed to install ginkgo: %v", err)
	}

	fmt.Println("\n✅ All dependencies installed successfully!")
	fmt.Printf("Binaries installed to: %s\n", binDir)
	
	// Verify installations
	fmt.Println("\nVerifying installations:")
	
	// Check luxd
	luxdPath := filepath.Join(binDir, "luxd")
	if _, err := os.Stat(luxdPath); err == nil {
		fmt.Printf("✓ luxd installed at %s\n", luxdPath)
	} else {
		fmt.Printf("✗ luxd not found at %s\n", luxdPath)
	}

	// Check lux
	luxPath := filepath.Join(binDir, "lux")
	if _, err := os.Stat(luxPath); err == nil {
		fmt.Printf("✓ lux installed at %s\n", luxPath)
	} else {
		fmt.Printf("✗ lux not found at %s\n", luxPath)
	}

	// Check ginkgo
	ginkgoPath := filepath.Join(binDir, "ginkgo")
	if _, err := os.Stat(ginkgoPath); err == nil {
		fmt.Printf("✓ ginkgo installed at %s\n", ginkgoPath)
	} else {
		fmt.Printf("✗ ginkgo not found at %s\n", ginkgoPath)
	}
}

func absPath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}