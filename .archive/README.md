# Scripts Directory

All legacy migration scripts have been archived.

Use the unified `genesis` tool instead:

```bash
# Build the tool
make build

# See all commands
./bin/genesis --help

# Import subnet as C-Chain
./bin/genesis import subnet <src> <dst>

# Launch with imported data
./bin/genesis launch L1
```

For the full pipeline:
```bash
make import-and-launch
```
