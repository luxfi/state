#!/usr/bin/env python3
"""
Efficient LUX NFT holder scanner for Ethereum
Gets current holders without scanning historical blocks
"""

import json
import asyncio
import aiohttp
from web3 import Web3
from collections import defaultdict
import time

# Ethereum RPC endpoints (you'll need to add your Infura/Alchemy key)
ETH_RPC = "https://mainnet.infura.io/v3/YOUR_INFURA_KEY"
# Fallback public endpoints (may have rate limits)
ETH_RPC_FALLBACKS = [
    "https://eth.llamarpc.com",
    "https://rpc.ankr.com/eth",
    "https://ethereum.publicnode.com"
]

# LUX NFT Contract on Ethereum
LUX_NFT_ADDRESS = "0x31e0f919c67cedd2bc3e294340dc900735810311"

# ERC721 ABI (minimal)
ERC721_ABI = [
    {
        "constant": True,
        "inputs": [],
        "name": "totalSupply",
        "outputs": [{"name": "", "type": "uint256"}],
        "type": "function"
    },
    {
        "constant": True,
        "inputs": [{"name": "tokenId", "type": "uint256"}],
        "name": "ownerOf",
        "outputs": [{"name": "", "type": "address"}],
        "type": "function"
    },
    {
        "constant": True,
        "inputs": [],
        "name": "name",
        "outputs": [{"name": "", "type": "string"}],
        "type": "function"
    }
]

async def scan_lux_nft_holders():
    """Main scanning function for LUX NFT holders"""
    print("Starting LUX NFT holder scan on Ethereum...")
    
    # Try to connect to RPC
    w3 = None
    rpcs = [ETH_RPC] + ETH_RPC_FALLBACKS
    
    for rpc in rpcs:
        try:
            w3 = Web3(Web3.HTTPProvider(rpc))
            if w3.is_connected():
                print(f"Connected to {rpc}")
                break
        except:
            continue
    
    if not w3 or not w3.is_connected():
        print("Failed to connect to Ethereum RPC")
        print("Please add your Infura/Alchemy key to the script")
        return None
    
    # Get contract
    nft_contract = w3.eth.contract(address=Web3.to_checksum_address(LUX_NFT_ADDRESS), abi=ERC721_ABI)
    
    # Get basic info
    try:
        name = nft_contract.functions.name().call()
        print(f"NFT Name: {name}")
    except:
        name = "LUX NFT"
    
    # Get total supply
    try:
        total_supply = nft_contract.functions.totalSupply().call()
        print(f"Total Supply: {total_supply}")
    except:
        print("Could not get total supply, trying enumeration...")
        # If totalSupply doesn't exist, we'll need to enumerate
        # For now, assume a reasonable max
        total_supply = 10000  # Adjust based on collection size
    
    # Get owners for each token
    holders = defaultdict(list)
    burned = []
    last_valid_token = 0
    consecutive_failures = 0
    
    print("\nScanning token owners...")
    
    for token_id in range(total_supply):
        if consecutive_failures > 100:  # Stop if we hit 100 non-existent tokens in a row
            print(f"Stopping at token {token_id} after 100 consecutive failures")
            break
        
        try:
            owner = nft_contract.functions.ownerOf(token_id).call()
            holders[owner.lower()].append(token_id)
            last_valid_token = token_id
            consecutive_failures = 0
            
            if token_id % 100 == 0:
                print(f"Processed {token_id} tokens...")
                
        except Exception as e:
            if "nonexistent token" in str(e).lower() or "invalid token" in str(e).lower():
                consecutive_failures += 1
            else:
                burned.append(token_id)
                consecutive_failures = 0
    
    # Adjust total supply to last valid token
    actual_total = last_valid_token + 1
    
    # Prepare results
    results = {
        "timestamp": int(time.time()),
        "contract_address": LUX_NFT_ADDRESS,
        "contract_name": name,
        "total_supply": actual_total,
        "unique_holders": len(holders),
        "burned_tokens": len(burned),
        "holders": {},
        "summary": {
            "total_nfts_held": actual_total - len(burned),
            "distribution": {},
            "validator_eligible": []  # Holders with 1+ NFT (eligible to be validators)
        }
    }
    
    # Format holder data and identify validator eligible
    for address, token_ids in holders.items():
        results["holders"][address] = {
            "token_count": len(token_ids),
            "token_ids": sorted(token_ids),
            "validator_eligible": True  # All NFT holders can be validators
        }
        results["summary"]["validator_eligible"].append(address)
    
    # Calculate distribution
    for address, data in results["holders"].items():
        count = data["token_count"]
        if count == 1:
            key = "1 NFT"
        elif count <= 5:
            key = "2-5 NFTs"
        elif count <= 10:
            key = "6-10 NFTs"
        else:
            key = "11+ NFTs"
        
        results["summary"]["distribution"][key] = results["summary"]["distribution"].get(key, 0) + 1
    
    return results

async def main():
    """Main entry point"""
    # Create output directory
    import os
    os.makedirs("exports/lux-analysis", exist_ok=True)
    
    results = await scan_lux_nft_holders()
    
    if not results:
        print("\nFailed to scan NFT holders. Please configure Ethereum RPC.")
        return
    
    # Save results
    with open("exports/lux-analysis/lux_nft_holders_current.json", "w") as f:
        json.dump(results, f, indent=2)
    
    # Generate CSV
    with open("exports/lux-analysis/lux_nft_holders_current.csv", "w") as f:
        f.write("address,token_count,token_ids,validator_eligible\n")
        for address, data in results["holders"].items():
            token_ids = ";".join(map(str, data["token_ids"]))
            f.write(f"{address},{data['token_count']},{token_ids},true\n")
    
    # Generate validator list
    with open("exports/lux-analysis/lux_validator_eligible.txt", "w") as f:
        f.write("# LUX NFT Holders - All Eligible to be Validators\n")
        f.write(f"# Total: {len(results['summary']['validator_eligible'])}\n\n")
        for address in sorted(results["summary"]["validator_eligible"]):
            f.write(f"{address}\n")
    
    # Print summary
    print("\n=== LUX NFT Holder Summary ===")
    print(f"Contract: {results['contract_name']} ({results['contract_address']})")
    print(f"Total Supply: {results['total_supply']}")
    print(f"Unique Holders: {results['unique_holders']}")
    print(f"Burned Tokens: {results['burned_tokens']}")
    print(f"Validator Eligible: {len(results['summary']['validator_eligible'])} addresses")
    
    print("\nDistribution:")
    for key, count in results['summary']['distribution'].items():
        print(f"  {key}: {count} holders")
    
    # Show top holders
    print("\nTop 10 LUX NFT Holders:")
    sorted_holders = sorted(results["holders"].items(), key=lambda x: x[1]["token_count"], reverse=True)
    for i, (address, data) in enumerate(sorted_holders[:10]):
        print(f"  {i+1}. {address}: {data['token_count']} NFTs")
    
    print(f"\nAll {results['unique_holders']} NFT holders are eligible to run validators!")

if __name__ == "__main__":
    asyncio.run(main())