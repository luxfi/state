package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

// PluginLoader handles loading and executing CLI plugins
type PluginLoader struct {
	PluginDir string
	RootCmd   *cobra.Command
}

// NewPluginLoader creates a new plugin loader
func NewPluginLoader(pluginDir string, rootCmd *cobra.Command) *PluginLoader {
	return &PluginLoader{
		PluginDir: pluginDir,
		RootCmd:   rootCmd,
	}
}

// LoadPlugins discovers and loads all plugins, adding them as commands
func (pl *PluginLoader) LoadPlugins() error {
	if _, err := os.Stat(pl.PluginDir); os.IsNotExist(err) {
		return nil // No plugin directory is ok
	}

	entries, err := os.ReadDir(pl.PluginDir)
	if err != nil {
		return fmt.Errorf("failed to read plugin directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			pluginPath := filepath.Join(pl.PluginDir, entry.Name())
			if err := pl.loadPlugin(pluginPath); err != nil {
				// Log warning but continue loading other plugins
				fmt.Fprintf(os.Stderr, "Warning: failed to load plugin %s: %v\n", entry.Name(), err)
			}
		}
	}

	return nil
}

// loadPlugin loads a single plugin and adds it as a command
func (pl *PluginLoader) loadPlugin(pluginPath string) error {
	// Look for plugin manifest
	manifestPath := filepath.Join(pluginPath, "plugin.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("plugin manifest not found")
	}

	var plugin Plugin
	if err := json.Unmarshal(data, &plugin); err != nil {
		return fmt.Errorf("invalid plugin manifest: %w", err)
	}

	// Make executable path absolute
	if !filepath.IsAbs(plugin.Executable) {
		plugin.Executable = filepath.Join(pluginPath, plugin.Executable)
	}

	// Verify executable exists
	if _, err := os.Stat(plugin.Executable); err != nil {
		return fmt.Errorf("plugin executable not found: %s", plugin.Executable)
	}

	// Create command for the plugin
	pluginCmd := &cobra.Command{
		Use:   plugin.Name,
		Short: plugin.Description,
		Long:  fmt.Sprintf("%s\n\nThis is a plugin command that extends lux-cli functionality.", plugin.Description),
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Execute the plugin with all arguments
			pluginExec := exec.Command(plugin.Executable, args...)
			pluginExec.Stdout = os.Stdout
			pluginExec.Stderr = os.Stderr
			pluginExec.Stdin = os.Stdin
			
			return pluginExec.Run()
		},
	}

	// Add the plugin command to the root command
	pl.RootCmd.AddCommand(pluginCmd)
	
	return nil
}

// Example usage in lux-cli main:
// func init() {
//     pluginLoader := cli.NewPluginLoader(filepath.Join(os.Getenv("HOME"), ".lux-cli/plugins"), rootCmd)
//     if err := pluginLoader.LoadPlugins(); err != nil {
//         fmt.Fprintf(os.Stderr, "Warning: failed to load plugins: %v\n", err)
//     }
// }