#!/usr/bin/env python3
"""
Efficient ZOO token burn scanner
Gets all burns to dead address by scanning Transfer events
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
ZOO_TOKEN_ADDRESS = "0x09e2b83fe5485a7c8beaa5dffd1d324a2b2d5c13"
DEAD_ADDRESS = "0x000000000000000000000000000000000000dEaD"

# Transfer event signature
TRANSFER_EVENT_SIGNATURE = Web3.keccak(text="Transfer(address,address,uint256)").hex()

async def scan_burns_in_range(w3, from_block, to_block):
    """Scan for burns in a specific block range"""
    try:
        logs = w3.eth.get_logs({
            'fromBlock': from_block,
            'toBlock': to_block,
            'address': Web3.to_checksum_address(ZOO_TOKEN_ADDRESS),
            'topics': [
                TRANSFER_EVENT_SIGNATURE,
                None,  # from address (any)
                Web3.keccak(text='0x' + DEAD_ADDRESS[2:].zfill(64)).hex()  # to address (dead)
            ]
        })
        return logs
    except Exception as e:
        # If range too large, split it
        if "limit exceeded" in str(e).lower() or "query returned more than" in str(e).lower():
            mid = (from_block + to_block) // 2
            logs1 = await scan_burns_in_range(w3, from_block, mid)
            logs2 = await scan_burns_in_range(w3, mid + 1, to_block)
            return logs1 + logs2
        else:
            print(f"Error scanning blocks {from_block}-{to_block}: {e}")
            return []

async def find_deployment_block(w3):
    """Binary search to find contract deployment block"""
    print("Finding ZOO token deployment block...")
    
    latest = w3.eth.block_number
    left, right = 0, latest
    
    while left < right:
        mid = (left + right) // 2
        
        try:
            code = w3.eth.get_code(Web3.to_checksum_address(ZOO_TOKEN_ADDRESS), block_identifier=mid)
            if code and len(code) > 2:  # Has contract code
                right = mid
            else:
                left = mid + 1
        except:
            left = mid + 1
    
    return left

async def scan_zoo_burns():
    """Main scanning function for ZOO burns"""
    print("Starting ZOO burn scan...")
    
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
    
    # Get current block
    latest_block = w3.eth.block_number
    print(f"Latest block: {latest_block}")
    
    # For ZOO, we know it was deployed around block 14-16M
    # To be efficient, let's start from block 14000000
    start_block = 14000000
    
    print(f"Scanning from block {start_block} to {latest_block}")
    print("This may take a few minutes...")
    
    # Scan for burns
    all_burns = []
    batch_size = 5000  # Adjust based on RPC limits
    
    for from_block in range(start_block, latest_block, batch_size):
        to_block = min(from_block + batch_size - 1, latest_block)
        
        # Rotate RPC for load balancing
        rpc = random.choice(RPC_ENDPOINTS)
        w3 = Web3(Web3.HTTPProvider(rpc))
        
        print(f"Scanning blocks {from_block:,} to {to_block:,}...")
        
        logs = await scan_burns_in_range(w3, from_block, to_block)
        all_burns.extend(logs)
        
        # Small delay to avoid rate limits
        await asyncio.sleep(0.1)
    
    print(f"\nFound {len(all_burns)} burn transactions")
    
    # Process burns
    burns_by_address = defaultdict(int)
    burn_details = []
    
    for log in all_burns:
        # Decode the log
        from_address = '0x' + log['topics'][1].hex()[26:]  # Remove padding
        amount = int(log['data'], 16)
        
        burns_by_address[from_address.lower()] += amount
        
        burn_details.append({
            'tx_hash': log['transactionHash'].hex(),
            'block_number': log['blockNumber'],
            'from': from_address.lower(),
            'amount': str(amount),
            'amount_decimal': amount / 10**18  # Assuming 18 decimals
        })
    
    # Prepare results
    results = {
        "timestamp": int(time.time()),
        "total_burns": len(all_burns),
        "unique_burners": len(burns_by_address),
        "total_burned_wei": str(sum(burns_by_address.values())),
        "total_burned_decimal": sum(burns_by_address.values()) / 10**18,
        "burns_by_address": {addr: str(amount) for addr, amount in burns_by_address.items()},
        "burn_details": burn_details[:1000]  # Limit details to first 1000
    }
    
    return results

async def main():
    """Main entry point"""
    results = await scan_zoo_burns()
    
    # Save results
    with open("exports/zoo-analysis/zoo_burns_complete.json", "w") as f:
        json.dump(results, f, indent=2)
    
    # Generate CSV summary
    with open("exports/zoo-analysis/zoo_burns_summary.csv", "w") as f:
        f.write("address,total_burned_wei,total_burned_decimal\n")
        for address, amount in results["burns_by_address"].items():
            decimal_amount = int(amount) / 10**18
            f.write(f"{address},{amount},{decimal_amount}\n")
    
    # Generate detailed CSV (limited)
    with open("exports/zoo-analysis/zoo_burns_details.csv", "w") as f:
        f.write("tx_hash,block_number,from_address,amount_wei,amount_decimal\n")
        for burn in results["burn_details"][:1000]:
            f.write(f"{burn['tx_hash']},{burn['block_number']},{burn['from']},{burn['amount']},{burn['amount_decimal']}\n")
    
    # Print summary
    print("\n=== ZOO Burn Summary ===")
    print(f"Total Burns: {results['total_burns']}")
    print(f"Unique Burners: {results['unique_burners']}")
    print(f"Total Burned: {results['total_burned_decimal']:,.2f} ZOO")
    
    # Show top burners
    print("\nTop 10 ZOO Burners:")
    sorted_burners = sorted(results["burns_by_address"].items(), key=lambda x: int(x[1]), reverse=True)
    for i, (address, amount) in enumerate(sorted_burners[:10]):
        decimal_amount = int(amount) / 10**18
        print(f"  {i+1}. {address}: {decimal_amount:,.2f} ZOO")

if __name__ == "__main__":
    asyncio.run(main())