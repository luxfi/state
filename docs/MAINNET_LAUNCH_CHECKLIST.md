# Lux Network Mainnet Launch Checklist

## Prerequisites ✅

### 1. Binaries Built
- [x] `luxd` at `../node/build/luxd`
- [x] `lux-cli` at `bin/lux-cli`
- [x] `genesis-builder` at `bin/genesis-builder`

### 2. Genesis Data Available
- [ ] C-Chain genesis from 96369: `data/unified-genesis/lux-mainnet-96369/genesis.json`
- [ ] C-Chain allocations: `data/unified-genesis/lux-mainnet-96369/allocations_combined.json`
- [ ] Zoo L2 genesis: `data/unified-genesis/zoo-mainnet-200200/genesis.json`
- [ ] SPC L2 genesis: `data/unified-genesis/spc-mainnet-36911/genesis.json`

### 3. Validator Configuration
- [ ] Real BLS public keys (48 bytes each) for 11 validators
- [ ] Real BLS proof of possession (96 bytes each) for 11 validators
- [ ] Create `configs/mainnet-validators-real.json` from template

## Launch Steps

### 1. Generate Real Validator Keys
```bash
# For each validator node, generate BLS keys
# This should be done on secure, air-gapped machines
luxd --generate-bls-key
```

### 2. Prepare Genesis
```bash
# Copy template and fill in real keys
cp configs/mainnet-validators-real.json.template configs/mainnet-validators-real.json
# Edit with real BLS keys

# Generate genesis with C-Chain import
./bin/genesis-builder \
    --network mainnet \
    --import-cchain data/unified-genesis/lux-mainnet-96369/genesis.json \
    --import-allocations data/unified-genesis/lux-mainnet-96369/allocations_combined.json \
    --validators configs/mainnet-validators-real.json \
    --output genesis_mainnet_96369.json
```

### 3. Launch Primary Network
```bash
# Using the launch script
./scripts/launch-mainnet.sh

# Or manually with lux-cli
./bin/lux-cli network start \
    --luxd-path ../node/build/luxd \
    --network-id 96369 \
    --custom-network-genesis genesis_mainnet_96369.json
```

### 4. Deploy L2 Subnets
```bash
# Zoo L2
./bin/lux-cli l2 create zoo \
    --evm \
    --chain-id 200200 \
    --custom-subnet-evm-genesis data/unified-genesis/zoo-mainnet-200200/genesis.json

./bin/lux-cli l2 deploy zoo --mainnet

# SPC L2
./bin/lux-cli l2 create spc \
    --evm \
    --chain-id 36911 \
    --custom-subnet-evm-genesis data/unified-genesis/spc-mainnet-36911/genesis.json

./bin/lux-cli l2 deploy spc --mainnet
```

## Network Configuration

### Primary Network (L1)
- Network ID: 96369
- Chain ID: 96369
- Bootstrap Nodes: 11 (52.53.185.222-232)
- Consensus: Snowman++
- Token: LUX (9 decimals on X/P, 18 on C)

### L2 Subnets
- **Zoo**: Chain ID 200200, Token: ZOO
- **SPC**: Chain ID 36911, Token: SPC
- **Hanzo**: Chain ID 36963, Token: AI (prepared, not deployed)

## Security Checklist

- [ ] All validator keys generated on secure machines
- [ ] Private keys properly secured
- [ ] Firewall rules configured
- [ ] Monitoring and alerting set up
- [ ] Backup procedures documented
- [ ] Incident response plan ready

## Monitoring

- [ ] Node health monitoring
- [ ] Network metrics dashboard
- [ ] Alert system configured
- [ ] Log aggregation set up

## Post-Launch

- [ ] Verify all validators are online
- [ ] Check consensus is working
- [ ] Test transactions on C-Chain
- [ ] Verify L2 subnet deployment
- [ ] Monitor network stability for 24 hours

## Emergency Procedures

- [ ] Rollback plan documented
- [ ] Emergency contact list
- [ ] Validator coordination channel
- [ ] Public communication plan

## Final Verification

Before launching mainnet, verify:
1. All 11 validators have real BLS keys
2. C-Chain genesis includes all 96369 data
3. Network security is properly configured
4. Monitoring and alerting are operational
5. All team members are ready

## Launch Command

When ready:
```bash
make launch-mainnet
```

This will execute the full mainnet launch sequence.