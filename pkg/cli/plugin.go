package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Plugin represents a CLI plugin that extends lux-cli functionality
type Plugin struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Executable  string            `json:"executable"`
	Commands    []PluginCommand   `json:"commands"`
}

// PluginCommand represents a command provided by a plugin
type PluginCommand struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Flags       []string `json:"flags"`
}

// PluginManager manages CLI plugins
type PluginManager struct {
	PluginDir string
	Plugins   map[string]*Plugin
}

// NewPluginManager creates a new plugin manager
func NewPluginManager(pluginDir string) *PluginManager {
	return &PluginManager{
		PluginDir: pluginDir,
		Plugins:   make(map[string]*Plugin),
	}
}

// LoadPlugins discovers and loads all plugins in the plugin directory
func (pm *PluginManager) LoadPlugins() error {
	if _, err := os.Stat(pm.PluginDir); os.IsNotExist(err) {
		return nil // No plugin directory is ok
	}

	entries, err := os.ReadDir(pm.PluginDir)
	if err != nil {
		return fmt.Errorf("failed to read plugin directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			pluginPath := filepath.Join(pm.PluginDir, entry.Name())
			if err := pm.loadPlugin(pluginPath); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to load plugin %s: %v\n", entry.Name(), err)
			}
		}
	}

	return nil
}

// loadPlugin loads a single plugin from a directory
func (pm *PluginManager) loadPlugin(pluginPath string) error {
	// Look for plugin manifest
	manifestPath := filepath.Join(pluginPath, "plugin.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		// Try executable with --help to auto-discover
		return pm.autoDiscoverPlugin(pluginPath)
	}

	var plugin Plugin
	if err := json.Unmarshal(data, &plugin); err != nil {
		return fmt.Errorf("invalid plugin manifest: %w", err)
	}

	// Make executable path absolute
	if !filepath.IsAbs(plugin.Executable) {
		plugin.Executable = filepath.Join(pluginPath, plugin.Executable)
	}

	pm.Plugins[plugin.Name] = &plugin
	return nil
}

// autoDiscoverPlugin tries to discover plugin capabilities by running --help
func (pm *PluginManager) autoDiscoverPlugin(pluginPath string) error {
	// Look for executable files
	entries, err := os.ReadDir(pluginPath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			execPath := filepath.Join(pluginPath, entry.Name())
			info, err := os.Stat(execPath)
			if err == nil && info.Mode()&0111 != 0 { // Is executable
				// Try running with --help
				cmd := exec.Command(execPath, "--help")
				output, err := cmd.Output()
				if err == nil {
					plugin := &Plugin{
						Name:        filepath.Base(pluginPath),
						Description: "Auto-discovered plugin",
						Executable:  execPath,
						Commands:    parseHelpOutput(string(output)),
					}
					pm.Plugins[plugin.Name] = plugin
					return nil
				}
			}
		}
	}

	return fmt.Errorf("no executable found in plugin directory")
}

// parseHelpOutput attempts to parse command information from help output
func parseHelpOutput(help string) []PluginCommand {
	// Simple parser - can be enhanced
	var commands []PluginCommand
	lines := strings.Split(help, "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Commands:") {
			// Start parsing commands section
			continue
		}
		// Add more sophisticated parsing here
	}
	
	return commands
}

// ExecutePlugin runs a plugin command
func (pm *PluginManager) ExecutePlugin(pluginName string, args []string) error {
	plugin, exists := pm.Plugins[pluginName]
	if !exists {
		return fmt.Errorf("plugin not found: %s", pluginName)
	}

	cmd := exec.Command(plugin.Executable, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

// ListPlugins returns information about all loaded plugins
func (pm *PluginManager) ListPlugins() []Plugin {
	var plugins []Plugin
	for _, plugin := range pm.Plugins {
		plugins = append(plugins, *plugin)
	}
	return plugins
}