package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PathConfig holds all path configuration for the genesis tool
type PathConfig struct {
	// Base paths
	ExecutableDir string // Directory containing the genesis executable
	WorkDir       string // Working directory (current directory or override)
	OutputDir     string // Output directory for generated files
	
	// Data paths
	ChaindataDir string // Directory containing blockchain data
	RuntimeDir   string // Directory for runtime/temporary data
	ConfigsDir   string // Directory for configuration files
	
	// File paths
	LuxdPath     string // Path to luxd binary
	LuxCLIPath   string // Path to lux-cli binary
}

// Global path configuration
var Paths *PathConfig

// InitializePaths sets up all path configuration based on:
// 1. Current working directory (default)
// 2. Environment variables
// 3. Command line flags
// 4. Saved settings in ~/.lux/genesis/
func InitializePaths() error {
	Paths = &PathConfig{}
	
	// Get executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	
	// Resolve symlinks
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}
	
	// Set executable directory
	Paths.ExecutableDir = filepath.Dir(execPath)
	
	// Default work directory is current working directory
	pwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}
	
	// Allow override via environment variable
	if workDir := os.Getenv("GENESIS_WORK_DIR"); workDir != "" {
		Paths.WorkDir = expandPath(workDir)
	} else {
		Paths.WorkDir = pwd
	}
	
	// Set output directory
	if outputDir := os.Getenv("GENESIS_OUTPUT_DIR"); outputDir != "" {
		Paths.OutputDir = expandPath(outputDir)
	} else {
		Paths.OutputDir = filepath.Join(Paths.WorkDir, "output")
	}
	
	// Set data directories
	Paths.ChaindataDir = filepath.Join(Paths.WorkDir, "chaindata")
	Paths.RuntimeDir = filepath.Join(Paths.WorkDir, "runs")
	Paths.ConfigsDir = filepath.Join(Paths.WorkDir, "configs")
	
	// Allow overrides
	if dir := os.Getenv("GENESIS_CHAINDATA_DIR"); dir != "" {
		Paths.ChaindataDir = expandPath(dir)
	}
	if dir := os.Getenv("GENESIS_RUNTIME_DIR"); dir != "" {
		Paths.RuntimeDir = expandPath(dir)
	}
	if dir := os.Getenv("GENESIS_CONFIGS_DIR"); dir != "" {
		Paths.ConfigsDir = expandPath(dir)
	}
	
	// Set tool paths - look in multiple locations
	Paths.LuxdPath = findTool("luxd", []string{
		filepath.Join(Paths.WorkDir, "..", "node", "build", "luxd"),
		filepath.Join(Paths.WorkDir, "node", "build", "luxd"),
		filepath.Join(os.Getenv("HOME"), "work", "lux", "node", "build", "luxd"),
		"/usr/local/bin/luxd",
		"luxd", // In PATH
	})
	
	Paths.LuxCLIPath = findTool("avalanche", []string{
		filepath.Join(Paths.WorkDir, "..", "cli", "bin", "avalanche"),
		filepath.Join(Paths.WorkDir, "cli", "bin", "avalanche"),
		filepath.Join(os.Getenv("HOME"), "work", "lux", "cli", "bin", "avalanche"),
		"/usr/local/bin/avalanche",
		"avalanche", // In PATH
	})
	
	// Allow tool path overrides
	if path := os.Getenv("LUXD_PATH"); path != "" {
		Paths.LuxdPath = expandPath(path)
	}
	if path := os.Getenv("LUX_CLI_PATH"); path != "" {
		Paths.LuxCLIPath = expandPath(path)
	}
	
	// Load saved settings (for tool paths mainly)
	if err := LoadSettings(); err != nil {
		// Ignore errors loading settings, just use defaults
		fmt.Fprintf(os.Stderr, "Warning: failed to load settings: %v\n", err)
	}
	
	return nil
}

// expandPath expands ~ and environment variables in a path
func expandPath(path string) string {
	// Expand ~ to home directory
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, path[2:])
	}
	
	// Expand environment variables
	path = os.ExpandEnv(path)
	
	// Make absolute if relative
	if !filepath.IsAbs(path) {
		path, _ = filepath.Abs(path)
	}
	
	return path
}

// findTool looks for a tool in multiple locations
func findTool(name string, locations []string) string {
	for _, loc := range locations {
		path := expandPath(loc)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return name // Return just the name, let exec.LookPath find it
}

// ResolvePath resolves a path relative to the work directory
func ResolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(Paths.WorkDir, path)
}

// SetCommandLinePaths updates paths based on command line flags
func SetCommandLinePaths(workDir, outputDir, chaindataDir string) {
	if workDir != "" {
		Paths.WorkDir = expandPath(workDir)
	}
	if outputDir != "" {
		Paths.OutputDir = expandPath(outputDir)
	}
	if chaindataDir != "" {
		Paths.ChaindataDir = expandPath(chaindataDir)
	}
}

// PrintPaths prints the current path configuration
func PrintPaths() {
	fmt.Println("Path Configuration:")
	fmt.Printf("  Executable Dir: %s\n", Paths.ExecutableDir)
	fmt.Printf("  Work Dir:       %s\n", Paths.WorkDir)
	fmt.Printf("  Output Dir:     %s\n", Paths.OutputDir)
	fmt.Printf("  Chaindata Dir:  %s\n", Paths.ChaindataDir)
	fmt.Printf("  Runtime Dir:    %s\n", Paths.RuntimeDir)
	fmt.Printf("  Configs Dir:    %s\n", Paths.ConfigsDir)
	fmt.Printf("  Luxd Path:      %s\n", Paths.LuxdPath)
	fmt.Printf("  Lux CLI Path:   %s\n", Paths.LuxCLIPath)
}

// GetSettingsPath returns the path to the settings file
func GetSettingsPath() string {
	home, _ := os.UserHomeDir()
	settingsDir := filepath.Join(home, ".lux", "genesis")
	return filepath.Join(settingsDir, "settings.json")
}

// SaveSettings saves the current path configuration to ~/.lux/genesis/settings.json
func SaveSettings() error {
	settingsPath := GetSettingsPath()
	settingsDir := filepath.Dir(settingsPath)
	
	// Create directory if it doesn't exist
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		return fmt.Errorf("failed to create settings directory: %w", err)
	}
	
	// Marshal settings to JSON
	data, err := json.MarshalIndent(Paths, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}
	
	// Write to file
	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write settings: %w", err)
	}
	
	return nil
}

// LoadSettings loads saved settings from ~/.lux/genesis/settings.json
func LoadSettings() error {
	settingsPath := GetSettingsPath()
	
	// Check if file exists
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		// No saved settings, use defaults
		return nil
	}
	
	// Read file
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return fmt.Errorf("failed to read settings: %w", err)
	}
	
	// Unmarshal settings
	saved := &PathConfig{}
	if err := json.Unmarshal(data, saved); err != nil {
		return fmt.Errorf("failed to unmarshal settings: %w", err)
	}
	
	// Apply saved settings (but don't override current directory defaults)
	// This allows settings to persist tool locations but not force work directories
	if Paths.LuxdPath == "luxd" && saved.LuxdPath != "" {
		Paths.LuxdPath = saved.LuxdPath
	}
	if Paths.LuxCLIPath == "lux-cli" && saved.LuxCLIPath != "" {
		Paths.LuxCLIPath = saved.LuxCLIPath
	}
	
	return nil
}