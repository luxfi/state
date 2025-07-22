#!/usr/bin/env python3
"""
Efficient EGG NFT holder scanner
Gets current holders without scanning historical blocks
"""

import json
import asyncio
import aiohttp
from web3 import Web3
from collections import defaultdict
import random
import time

# Multiple BSC RPC endpoints for load balancing
RPC_ENDPOINTS = [
    "https://bsc-dataseed.bnbchain.org",
    "https://bsc-dataseed.nariox.org", 
    "https://bsc-dataseed.defibit.io",
    "https://bsc-dataseed.ninicoin.io",
    "https://bsc-dataseed1.binance.org",
    "https://bsc-dataseed2.binance.org",
    "https://bsc-dataseed3.binance.org",
    "https://bsc-dataseed4.binance.org"
]

# Contract addresses
EGG_NFT_ADDRESS = "0x5bb68cf06289d54efde25155c88003be685356a8"
ZOO_TOKEN_ADDRESS = "0x09e2b83fe5485a7c8beaa5dffd1d324a2b2d5c13"

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
    }
]

# ERC20 ABI (minimal)
ERC20_ABI = [
    {
        "constant": True,
        "inputs": [{"name": "_owner", "type": "address"}],
        "name": "balanceOf",
        "outputs": [{"name": "balance", "type": "uint256"}],
        "type": "function"
    }
]

async def get_owner_batch(session, rpc_url, contract, token_ids):
    """Get owners for a batch of token IDs"""
    w3 = Web3(Web3.HTTPProvider(rpc_url))
    nft_contract = w3.eth.contract(address=Web3.to_checksum_address(EGG_NFT_ADDRESS), abi=ERC721_ABI)
    
    results = {}
    for token_id in token_ids:
        try:
            owner = nft_contract.functions.ownerOf(token_id).call()
            results[token_id] = owner.lower()
        except Exception as e:
            # Token might be burned or doesn't exist
            results[token_id] = None
    
    return results

async def get_zoo_balance(session, rpc_url, address):
    """Get ZOO token balance for an address"""
    try:
        w3 = Web3(Web3.HTTPProvider(rpc_url))
        zoo_contract = w3.eth.contract(address=Web3.to_checksum_address(ZOO_TOKEN_ADDRESS), abi=ERC20_ABI)
        balance = zoo_contract.functions.balanceOf(Web3.to_checksum_address(address)).call()
        return balance
    except:
        return 0

async def scan_egg_holders():
    """Main scanning function"""
    print("Starting EGG NFT holder scan...")
    
    # Get a working RPC
    w3 = None
    for rpc in RPC_ENDPOINTS:
        try:
            w3 = Web3(Web3.HTTPProvider(rpc))
            if w3.is_connected():
                print(f"Connected to {rpc}")
                break
        except:
            continue
    
    if not w3 or not w3.is_connected():
        print("Failed to connect to any RPC")
        return
    
    # Get total supply
    nft_contract = w3.eth.contract(address=Web3.to_checksum_address(EGG_NFT_ADDRESS), abi=ERC721_ABI)
    try:
        total_supply = nft_contract.functions.totalSupply().call()
        print(f"Total EGG NFTs: {total_supply}")
    except:
        print("Using known total supply: 145")
        total_supply = 145
    
    # Get owners for each token
    holders = defaultdict(list)
    burned = []
    
    print("\nScanning token owners...")
    batch_size = 10
    
    async with aiohttp.ClientSession() as session:
        for i in range(1, total_supply + 1, batch_size):
            # Pick a random RPC for load balancing
            rpc = random.choice(RPC_ENDPOINTS)
            
            # Get batch of token IDs
            batch = list(range(i, min(i + batch_size, total_supply + 1)))
            
            try:
                results = await get_owner_batch(session, rpc, nft_contract, batch)
                
                for token_id, owner in results.items():
                    if owner:
                        holders[owner].append(token_id)
                    else:
                        burned.append(token_id)
                
                print(f"Processed tokens {i} to {min(i + batch_size - 1, total_supply)}")
                
                # Small delay to avoid rate limits
                await asyncio.sleep(0.1)
                
            except Exception as e:
                print(f"Error processing batch {i}: {e}")
                # Fallback to sequential processing for this batch
                for token_id in batch:
                    try:
                        owner = nft_contract.functions.ownerOf(token_id).call()
                        holders[owner.lower()].append(token_id)
                    except:
                        burned.append(token_id)
    
    # Get ZOO balances for holders
    print("\nGetting ZOO balances for holders...")
    zoo_balances = {}
    
    async with aiohttp.ClientSession() as session:
        for address in holders.keys():
            rpc = random.choice(RPC_ENDPOINTS)
            balance = await get_zoo_balance(session, rpc, address)
            zoo_balances[address] = str(balance)
            await asyncio.sleep(0.05)  # Rate limit
    
    # Prepare results
    results = {
        "timestamp": int(time.time()),
        "total_supply": total_supply,
        "unique_holders": len(holders),
        "burned_tokens": len(burned),
        "holders": {},
        "summary": {
            "total_eggs_held": total_supply - len(burned),
            "zoo_equivalent": (total_supply - len(burned)) * 4200000,
            "distribution": {}
        }
    }
    
    # Format holder data
    for address, token_ids in holders.items():
        results["holders"][address] = {
            "token_count": len(token_ids),
            "token_ids": sorted(token_ids),
            "zoo_equivalent": len(token_ids) * 4200000,
            "current_zoo_balance": zoo_balances.get(address, "0")
        }
    
    # Calculate distribution
    for address, data in results["holders"].items():
        count = data["token_count"]
        if count == 1:
            key = "1 egg"
        elif count <= 5:
            key = "2-5 eggs"
        elif count <= 10:
            key = "6-10 eggs"
        else:
            key = "11+ eggs"
        
        results["summary"]["distribution"][key] = results["summary"]["distribution"].get(key, 0) + 1
    
    return results

async def main():
    """Main entry point"""
    results = await scan_egg_holders()
    
    # Save results
    with open("exports/zoo-analysis/egg_holders_current.json", "w") as f:
        json.dump(results, f, indent=2)
    
    # Generate CSV
    with open("exports/zoo-analysis/egg_holders_current.csv", "w") as f:
        f.write("address,token_count,token_ids,zoo_equivalent,current_zoo_balance\n")
        for address, data in results["holders"].items():
            token_ids = ";".join(map(str, data["token_ids"]))
            f.write(f"{address},{data['token_count']},{token_ids},{data['zoo_equivalent']},{data['current_zoo_balance']}\n")
    
    # Print summary
    print("\n=== EGG NFT Holder Summary ===")
    print(f"Total Supply: {results['total_supply']}")
    print(f"Unique Holders: {results['unique_holders']}")
    print(f"Burned Tokens: {results['burned_tokens']}")
    print(f"Total ZOO Equivalent: {results['summary']['zoo_equivalent']:,}")
    print("\nDistribution:")
    for key, count in results['summary']['distribution'].items():
        print(f"  {key}: {count} holders")
    
    # Show top holders
    print("\nTop 10 EGG Holders:")
    sorted_holders = sorted(results["holders"].items(), key=lambda x: x[1]["token_count"], reverse=True)
    for i, (address, data) in enumerate(sorted_holders[:10]):
        print(f"  {i+1}. {address}: {data['token_count']} eggs ({data['zoo_equivalent']:,} ZOO)")

if __name__ == "__main__":
    asyncio.run(main())