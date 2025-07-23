# Treasury Balance Verification

This guide shows how to verify the treasury account balance to ensure the integrity of the network genesis.

## Treasury Account

The canonical treasury account is:

```
0x9011E888251AB053B7bD1cdB598Db4f9DEd94714
```

It is initialized with 2T LUX in the genesis. After accounting for real usage, the mainnet balance should be approximately 1.995T LUX, and the testnet balance should be below 1.9T LUX.

## Prerequisites

- `denamespace` (built via `make install-plugin` or `make build-tools`)
- `evmarchaeology` (built via `make install-plugin` or `make build-tools`)

## Steps

1. **Extract C-Chain State**

```bash
denamespace \
  -src chaindata/lux-mainnet/96369/db/pebbledb \
  -dst /tmp/extracted-96369 \
  -network 96369 \
  -state
```

2. **Analyze with evmarchaeology**

```bash
evmarchaeology analyze \
  -db /tmp/extracted-96369 \
  -account 0x9011E888251AB053B7bD1cdB598Db4f9DEd94714
```

3. **Compare the Result**

Expected output (approximate):

```text
Account: 0x9011E888251AB053B7bD1cdB598Db4f9DEd94714
Balance: 1.995000000000000000 T LUX
```

For testnet (chain ID 96368), repeat with the corresponding path:

```bash
denamespace \
  -src chaindata/lux-testnet-96368/db/pebbledb \
  -dst /tmp/extracted-96368 \
  -network 96368 \
  -state

evmarchaeology analyze \
  -db /tmp/extracted-96368 \
  -account 0x9011E888251AB053B7bD1cdB598Db4f9DEd94714
```

Expected balance: < 1.9 T LUX.

## Troubleshooting

- Ensure the `chaindata` path points to a valid PebbleDB directory.
- Rebuild tools with `make build-tools` if binaries are missing.
