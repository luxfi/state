# LUX Subnet to C-Chain Migration Summary

## Overview

This document summarizes the progress on migrating LUX subnet data (chain 96369) to C-Chain format for use with luxd.

## Key Findings

### 1. Database Structure
- **Subnet Format**: Uses 33-byte namespace prefix + logical key + 8-byte revision suffix (VersionDB wrapper)
- **Total Keys**: ~34 million keys in LUX mainnet archive
- **Key Types Found**:
  - Headers (h): 227,986
  - Bodies (b): 228,888  
  - Receipts (r): 228,230
  - Hash->Number (H): 228,568
  - Number->Hash (n): 114,661
  - Accounts: 113,779
  - Storage: ~10 million

### 2. Critical Issues Discovered

#### Truncated Hash 'n' Keys
- Subnet 'n' keys use format: `n + 22-byte-truncated-hash` instead of `n + 8-byte-number`
- Only 4,285 'H' keys available to map truncated hashes to block numbers
- 114,661 'n' keys cannot be matched (0% success rate)
- This prevents creation of proper canonical mappings required by C-Chain

#### Treasury Account Status
- Treasury address: `0x9011e888251ab053b7bd1cdb598db4f9ded94714`
- **NOT FOUND** in the account keys (checked all 113,779 accounts)
- Found in transaction data but not as a state account
- Expected balance: >1.9T LUX

### 3. Migration Pipeline Status

#### Completed Components
1. ✅ **migrate_evm** - Strips namespace prefix and adds "evm" prefix
2. ✅ **Caching system** - Reduces repeat processing from >2min to <1sec
3. ✅ **Ginkgo test suite** - Comprehensive testing framework
4. ✅ **Mini-lab test harness** - 90-second sanity check
5. ✅ **Key format analysis tools** - Deep inspection of database structure

#### Blocked Components
1. ❌ **evmn key fixing** - Cannot create proper mappings due to truncated hashes
2. ❌ **Consensus replay** - Requires canonical mappings to work
3. ❌ **RPC verification** - Node shows block 0 without proper migration
4. ❌ **Treasury balance check** - Account not found in state data

## Technical Details

### Migration Process
```bash
# Step 1: Migrate keys (working)
bin/migrate_evm --src <subnet-db> --dst <evm-db>

# Step 2: Fix canonical mappings (BLOCKED)
# Cannot match truncated hashes to block numbers

# Step 3: Replay consensus (BLOCKED)
bin/replay-consensus-pebble --evm <evm-db> --state <state-db> --tip <height>

# Step 4: Launch luxd (shows block 0)
luxd --db-dir <migrated-db> --network-id 96369
```

### Root Cause Analysis
The subnet data appears to be from a different version or configuration that:
1. Uses truncated hashes in canonical mappings
2. Doesn't include complete hash->number mappings
3. May not have the treasury account in the expected format

## Recommendations

### Immediate Actions
1. **Verify Data Source**: Confirm this is the correct and complete archive
2. **Check Alternative Formats**: Treasury might be stored differently in subnet
3. **Expand H Key Search**: Look for additional hash->number mappings

### Long-term Solutions
1. **Reconstruct Mappings**: Build 'n' keys from header data if possible
2. **Import From Different Source**: Get data from a node with complete mappings
3. **Custom Migration Tool**: Handle the specific truncated hash format

## Test Results

### ZOO Subnet (Test Data)
- 475 blocks
- Only 1 hash->number mapping
- 62 'n' keys, 0 matched
- Similar truncated hash issue

### LUX Mainnet
- Expected: 1,082,781 blocks
- Found: Headers/bodies for ~228k blocks
- No working canonical mappings
- Treasury account missing

## Conclusion

The migration pipeline is technically sound but the source data has unexpected format differences that prevent successful migration. The key blocker is the truncated hash format in 'n' keys combined with insufficient hash->number mappings to reconstruct proper canonical keys.

## Next Steps

1. Investigate alternative data sources
2. Analyze transaction data for treasury movements
3. Consider building custom tools for this specific subnet format
4. Consult with subnet operators about the data format

---

Generated: 2025-07-28
Tools Version: genesis/v2.0