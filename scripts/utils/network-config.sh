#!/bin/bash

# Network configuration for luxd

NETWORK="${1:-mainnet}"

case "$NETWORK" in
    mainnet)
        NETWORK_ID=96369
        CHAIN_ID=96369
        BOOTSTRAP_IPS="52.53.185.222:9651,52.53.185.223:9651,52.53.185.224:9651,52.53.185.225:9651,52.53.185.226:9651,52.53.185.227:9651,52.53.185.228:9651,52.53.185.229:9651,52.53.185.230:9651,52.53.185.231:9651,52.53.185.232:9651"
        BOOTSTRAP_IDS="NodeID-Mp8JrhoLmrGznZoYsszM19W6dTdcR35NF,NodeID-Nf5M5YoDN5CfR1wEmCPsf5zt2ojTZZj6j,NodeID-JCBCEeyRZdeDxEhwoztS55fsWx9SwJDVL,NodeID-JQvVo8DpzgyjhEDZKgqsFLVUPmN6JP3ig,NodeID-PKTUGFE6jnQbnskSDM3zvmQjnHKV3fxy4,NodeID-LtBrcgdgPW9Nj9JoU1AwGeCgi29R9JoQC,NodeID-962omv3YgJsqbcPvVR4yDHU8RPtaKCLt,NodeID-LPznW4BxjJaFYP5KEuJUenwVGTkH48XDe,NodeID-4nDStCMacNr5aadavMZxAxk9m9bfFf69F,NodeID-GGpbeWwfsZBaasex25ZPMkJFN713BXx7u,NodeID-Fh7dFdzt1QYQDTKJfZTVBLMyPipP99AmH"
        STAKING_ENABLED=true
        SYBIL_PROTECTION_ENABLED=true
        SNOW_SAMPLE_SIZE=20
        SNOW_QUORUM_SIZE=14
        SNOW_VIRTUOUS_COMMIT_THRESHOLD=14
        SNOW_ROGUE_COMMIT_THRESHOLD=20
        ;;
        
    testnet)
        NETWORK_ID=96368
        CHAIN_ID=96368
        BOOTSTRAP_IPS="testnet1.lux.network:9651,testnet2.lux.network:9651"
        BOOTSTRAP_IDS="NodeID-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx,NodeID-yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy"
        STAKING_ENABLED=true
        SYBIL_PROTECTION_ENABLED=true
        SNOW_SAMPLE_SIZE=5
        SNOW_QUORUM_SIZE=3
        SNOW_VIRTUOUS_COMMIT_THRESHOLD=3
        SNOW_ROGUE_COMMIT_THRESHOLD=5
        ;;
        
    local)
        NETWORK_ID=12345
        CHAIN_ID=12345
        BOOTSTRAP_IPS=""
        BOOTSTRAP_IDS=""
        STAKING_ENABLED=false
        SYBIL_PROTECTION_ENABLED=false
        SNOW_SAMPLE_SIZE=1
        SNOW_QUORUM_SIZE=1
        SNOW_VIRTUOUS_COMMIT_THRESHOLD=1
        SNOW_ROGUE_COMMIT_THRESHOLD=1
        ;;
        
    local-poa)
        # POA configuration for single-node automining
        NETWORK_ID=96369
        CHAIN_ID=96369
        BOOTSTRAP_IPS=""
        BOOTSTRAP_IDS=""
        STAKING_ENABLED=false
        SYBIL_PROTECTION_ENABLED=false
        SNOW_SAMPLE_SIZE=1
        SNOW_QUORUM_SIZE=1
        SNOW_VIRTUOUS_COMMIT_THRESHOLD=1
        SNOW_ROGUE_COMMIT_THRESHOLD=1
        ;;
        
    *)
        echo "Unknown network: $NETWORK"
        echo "Valid networks: mainnet, testnet, local, local-poa"
        exit 1
        ;;
esac

# Export all variables
export NETWORK_ID CHAIN_ID BOOTSTRAP_IPS BOOTSTRAP_IDS
export STAKING_ENABLED SYBIL_PROTECTION_ENABLED
export SNOW_SAMPLE_SIZE SNOW_QUORUM_SIZE
export SNOW_VIRTUOUS_COMMIT_THRESHOLD SNOW_ROGUE_COMMIT_THRESHOLD