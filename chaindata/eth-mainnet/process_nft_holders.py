#!/usr/bin/env python3
import json
import sys
from collections import defaultdict

# Load transfer events
with open('lux_nft_transfer_events.json', 'r') as f:
    transfers = json.load(f)

# Track NFT ownership
nft_owners = {}  # tokenId -> current owner
owner_tokens = defaultdict(list)  # owner -> list of tokenIds

# Process transfers chronologically
for transfer in transfers:
    if len(transfer['topics']) >= 4:
        from_addr = '0x' + transfer['topics'][1][-40:]
        to_addr = '0x' + transfer['topics'][2][-40:]
        token_id = int(transfer['topics'][3], 16)
        
        # Remove from previous owner
        if token_id in nft_owners:
            prev_owner = nft_owners[token_id]
            if token_id in owner_tokens[prev_owner]:
                owner_tokens[prev_owner].remove(token_id)
        
        # Add to new owner (unless burning to 0x0)
        if to_addr != '0x0000000000000000000000000000000000000000':
            nft_owners[token_id] = to_addr
            owner_tokens[to_addr].append(token_id)

# Create holder summary
holders = []
for owner, tokens in owner_tokens.items():
    if tokens:  # Only include if they still own tokens
        holders.append({
            'address': owner,
            'tokenIds': sorted(tokens),
            'tokenCount': len(tokens)
        })

# Sort by token count
holders.sort(key=lambda x: x['tokenCount'], reverse=True)

# Save results
with open('lux_nft_current_holders.json', 'w') as f:
    json.dump({
        'snapshot_block': FINALIZED_BLOCK,
        'total_holders': len(holders),
        'total_nfts': len(nft_owners),
        'holders': holders
    }, f, indent=2)

# Create CSV
with open('lux_nft_holders.csv', 'w') as f:
    f.write('address,token_count,token_ids\n')
    for holder in holders:
        token_ids_str = ';'.join(map(str, holder['tokenIds']))
        f.write(f"{holder['address']},{holder['tokenCount']},{token_ids_str}\n")

print(f"Found {len(holders)} NFT holders with {len(nft_owners)} total NFTs")
