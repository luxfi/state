#!/bin/bash

echo "Testing dev node..."
echo ""
echo "Network ID:"
curl -s -X POST -H 'Content-Type: application/json' \
    -d '{"jsonrpc":"2.0","id":1,"method":"info.getNetworkID","params":{}}' \
    http://localhost:9630/ext/info | jq .

echo ""
echo "C-Chain ID:"
curl -s -X POST -H 'Content-Type: application/json' \
    -d '{"jsonrpc":"2.0","id":1,"method":"eth_chainId","params":[]}' \
    http://localhost:9630/ext/bc/C/rpc | jq .

echo ""
echo "Block Number:"
curl -s -X POST -H 'Content-Type: application/json' \
    -d '{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}' \
    http://localhost:9630/ext/bc/C/rpc | jq .