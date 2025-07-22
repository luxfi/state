# 7777 Chain Data Extraction Guide

## Prerequisites

- December 2023 Avalanche v1.10.17 database backup
- Go 1.21+ 
- 20GB free disk space

## Extraction Process

### Step 1: Locate Chain Data

Use the prefix scanner to find the 7777 chain data:

```bash
cd scripts/2023-7777
go build prefixscan.go
./prefixscan -db /path/to/leveldb -pattern 1e61
```

Expected output:
```
Prefix: 0d3632155dd8689d7188dc377becf8265c1ef9e24c76a70f28015ab0c299840b33
```

### Step 2: Convert Database

```bash
go build convert-7777-specific.go
./convert-7777-specific \
    -src /path/to/leveldb \
    -dst ../../data/2023-7777/pebble-clean \
    -v
```

### Step 3: Verify Conversion

```bash
# Check output size (~441MB)
du -sh ../../data/2023-7777/pebble-clean/
```

## Output

- **Location**: `data/2023-7777/pebble-clean/`
- **Format**: PebbleDB (ready for import)
- **Keys**: 6,035,270
- **Size**: ~441MB