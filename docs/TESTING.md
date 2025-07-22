# Genesis Testing Guide

This guide explains how to run the comprehensive test suite for the LUX Genesis project.

## Prerequisites

1. **Go 1.24.5** - Required for building and testing
2. **Install binaries**: Run `make install` to download:
   - `bin/luxd` - LUX node binary
   - `bin/lux` - LUX CLI tool
   - `bin/ginkgo` - Test runner
3. **Chain data** in `chaindata/` directory
4. **At least 20GB free disk space**

## Test Structure

```
tests/
├── genesis_suite_test.go    # Main test suite setup
├── database_test.go         # Database validation tests
└── integration/
    └── network_test.go      # Full integration tests
```

## Running Tests

### Install Test Dependencies
```bash
make install-test-deps
```

### Run Unit Tests
```bash
make test-unit
```

### Run Integration Tests
```bash
make test-integration
```

### Run All Tests
```bash
make test-all
```

### Run Full Integration Test
This runs the complete workflow:
```bash
make test-full-integration
```

## Test Coverage

The test suite covers:

1. **Database Operations**
   - LevelDB to PebbleDB conversion
   - Data integrity validation
   - Key prefix verification
   - Genesis block validation

2. **Network Setup**
   - 5-node primary network creation
   - Network health verification
   - Node status checking

3. **C-Chain Import**
   - Historic data import (7777 and 96369)
   - RPC endpoint verification
   - Block number validation

4. **L2 Subnet Deployment**
   - ZOO subnet (chain ID 200200)
   - SPC subnet (chain ID 36911)
   - Hanzo subnet (chain ID 36963)

5. **Dev Mode Testing**
   - 7777 chain in single-node mode
   - POA configuration validation
   - Chain ID verification

## Running Specific Tests

### Test only database operations
```bash
ginkgo -v --focus="Database Operations" tests/
```

### Test only network setup
```bash
ginkgo -v --focus="5-Node Primary Network" tests/integration/
```

### Test only 7777 dev mode
```bash
ginkgo -v --focus="7777 Dev Mode" tests/integration/
```

## Manual Testing

### Start Local Network
```bash
go run scripts/run_local_network.go -nodes 5
```

### Start Network with Data Import
```bash
go run scripts/run_local_network.go -nodes 5 -with-data -data-path pebbledb/lux-96369
```

### Start Network with L2s
```bash
go run scripts/run_local_network.go -nodes 5 -with-l2s
```

## Debugging Failed Tests

### Check logs
```bash
# Node logs
tail -f ~/.luxd/logs/main.log
tail -f ~/.luxd/logs/C.log

# CLI logs
tail -f ~/.avalanche-cli/logs/
```

### Verify binaries
```bash
# Check luxd version
~/work/lux/node/build/luxd --version

# Check CLI version
~/work/lux/cli/bin/avalanche --version
```

### Clean test data
```bash
# Remove test networks
rm -rf ~/.avalanche-cli/networks/test-*

# Clean PebbleDB conversions
rm -rf pebbledb/
```

## Continuous Integration

The test suite is designed to run in CI environments:

```yaml
# Example GitHub Actions workflow
- name: Run Genesis Tests
  run: |
    make install-test-deps
    make test-all
```

## Performance Considerations

- Database conversion tests may take 5-10 minutes
- Network startup tests require ~2 minutes per test
- Full integration test suite takes ~30 minutes
- Use `--timeout` flag for longer operations:
  ```bash
  ginkgo -v --timeout=60m tests/integration/
  ```

## Troubleshooting

### Test timeouts
Increase timeout in test files or command line:
```go
Eventually(condition, 5*time.Minute).Should(Succeed())
```

### Port conflicts
Ensure no other services are using:
- 9650 (RPC port)
- 9651 (Staking port)

### Database locks
Remove LOCK files if tests fail:
```bash
find chaindata/ -name "LOCK" -delete
find pebbledb/ -name "LOCK" -delete
```