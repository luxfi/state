# Genesis Plugin Integration for lux-cli

This document describes how the genesis functionality has been integrated as a plugin for lux-cli.

## Overview

The genesis tool has been packaged as a CLI plugin that extends lux-cli with genesis generation and blockchain data import capabilities.

## Plugin Structure

```
~/.lux-cli/plugins/genesis/
├── genesis            # The actual genesis binary
├── lux-cli-genesis   # Wrapper script
└── plugin.json       # Plugin manifest
```

## Installation

```bash
# Build and install the plugin
make build-genesis
make install-plugin
```

## Usage

Once installed, the genesis plugin can be used in two ways:

### 1. Direct Plugin Usage
```bash
# From the plugin directory
~/.lux-cli/plugins/genesis/lux-cli-genesis generate --network mainnet
```

### 2. Through lux-cli (requires integration)
```bash
# Once lux-cli integrates the plugin loader
lux-cli genesis generate --network mainnet
lux-cli genesis import historic --chain-data ./chaindata
```

## Plugin Manifest Format

The `plugin.json` file describes the plugin:

```json
{
  "name": "genesis",
  "description": "Genesis generation and blockchain data import tools",
  "executable": "./lux-cli-genesis",
  "commands": [
    {
      "name": "generate",
      "description": "Generate genesis files for Lux networks",
      "flags": ["--network", "--output-dir", "--validators-file"]
    },
    {
      "name": "import",
      "description": "Import historical blockchain data",
      "flags": ["--chain-data", "--network-id"]
    }
  ]
}
```

## Integration with lux-cli

To enable plugin support in lux-cli, add the following to the main command initialization:

```go
import "github.com/luxfi/genesis/pkg/cli"

func init() {
    pluginDir := filepath.Join(os.Getenv("HOME"), ".lux-cli/plugins")
    pluginLoader := cli.NewPluginLoader(pluginDir, rootCmd)
    if err := pluginLoader.LoadPlugins(); err != nil {
        fmt.Fprintf(os.Stderr, "Warning: failed to load plugins: %v\n", err)
    }
}
```

## Available Commands

### Generate Genesis
```bash
# Generate mainnet genesis with validators
lux-cli genesis generate --network mainnet --validators-file configs/mainnet-validators.json

# Generate testnet genesis
lux-cli genesis generate --network testnet --output-dir output-testnet
```

### Import Historic Data
```bash
# Import 96369 C-Chain data
lux-cli genesis import historic --chain-data ./chaindata --network-id 96369

# Import Zoo L2 data
lux-cli genesis import historic --chain-data ./chaindata --network-id 200200
```

## Deployment Commands

The plugin integrates with the make deployment system:

```bash
# Deploy mainnet with historical data
make deploy-mainnet

# Deploy testnet
make deploy-testnet

# Deploy local test network
make deploy-local
```

## Benefits of Plugin Architecture

1. **Modularity**: Genesis functionality is separate from core lux-cli
2. **Extensibility**: Easy to add new plugins for other functionality
3. **Version Independence**: Plugins can be updated without rebuilding lux-cli
4. **User Customization**: Users can add their own plugins
5. **Clean Separation**: Genesis-specific code doesn't bloat the main CLI

## Future Enhancements

1. **Auto-discovery**: Automatically discover and load plugins on startup
2. **Plugin Repository**: Central repository for community plugins
3. **Version Management**: Plugin versioning and compatibility checks
4. **Help Integration**: Merge plugin help into main lux-cli help
5. **Configuration**: Per-plugin configuration files

## Technical Details

The plugin system uses a generic executable wrapper that:
- Reads plugin manifests
- Validates plugin executables
- Forwards commands and arguments
- Preserves stdin/stdout/stderr for seamless integration

This approach allows any executable to become a lux-cli plugin with just a manifest file.