#!/usr/bin/env bash
set -euo pipefail
ROOT=$(mktemp -d)               # /tmp/tmp.xxxxx
SRC=/home/z/archived/restored-blockchain-data/chainData/dnmzhuf6poM6PUNQCe7MWWfBdTJEnddhHRNXz2x7H6qSmyBEJ/db/pebbledb

echo "➜  Mini-lab root: $ROOT"

# ---------------------------------------------------------------------
# 1.  Copy only the SST files that contain the first ~100 heights
#     (they are always the lowest-numbered *.sst files).
# ---------------------------------------------------------------------
mkdir -p $ROOT/src/pebbledb
# pick first 8 SSTs (~40 MB) – adjust if your files are larger/smaller
ls $SRC/*.sst | sort | head -8 | xargs -I{} cp {} $ROOT/src/pebbledb/
# also copy the OPTIONS & MANIFEST so Pebble opens cleanly
cp $SRC/OPTIONS* $SRC/MANIFEST-* $ROOT/src/pebbledb/

# ---------------------------------------------------------------------
# 2.  Translate keys into Coreth layout  ➜  $ROOT/evm/pebbledb
# ---------------------------------------------------------------------
bin/migrate_evm \
    --src  $ROOT/src/pebbledb \
    --dst  $ROOT/evm/pebbledb \
    --verbose

# ---------------------------------------------------------------------
# 3.  Determine tip height only from our sample (peek_tip helper).
#     (Use a tiny, inlined Go one-liner so we don't depend on another exe.)
# ---------------------------------------------------------------------
TIP=$(go run - <<'EOF' $ROOT/evm/pebbledb
package main; import (
 "encoding/binary"; "fmt"; "os"; "github.com/cockroachdb/pebble")
func main(){
 db,_:=pebble.Open(os.Args[1],nil); it:=db.NewIter(nil); max:=uint64(0)
 p:=append([]byte("evm"),'n')      // evmn
 for it.SeekGE(p); it.Valid() && len(it.Key())>=12 && string(it.Key()[:4])=="evmn"; it.Next(){
  n:=binary.BigEndian.Uint64(it.Key()[4:12]); if n>max{max=n}
 }
 fmt.Print(max)
}
EOF
)
echo "➜  Sample tip height  = $TIP"

# ---------------------------------------------------------------------
# 4.  Replay consensus into VersionDB  ➜  $ROOT/state/pebbledb
# ---------------------------------------------------------------------
bin/replay-consensus-pebble \
    --evm   $ROOT/evm/pebbledb \
    --state $ROOT/state/pebbledb \
    --tip   "$TIP"

# ---------------------------------------------------------------------
# 5.  Launch a disposable luxd that uses the two sub-DBs.
# ---------------------------------------------------------------------
PORT=9655
luxd \
  --db-dir              $ROOT \
  --network-id          96369 \
  --staking-enabled=false \
  --http-port           $PORT \
  --log-level           info \
  --chain-configs.enable-indexing &
NODEPID=$!
echo "➜  luxd PID = $NODEPID"
sleep 6   # give VM time to initialise

# ---------------------------------------------------------------------
# 6.  Verify eth_blockNumber matches our TIP
# ---------------------------------------------------------------------
HEXTIP=$(printf '0x%x' "$TIP")
BN=$(curl -s --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
           http://127.0.0.1:$PORT/ext/bc/C/rpc | jq -r .result)
echo "➜  RPC eth_blockNumber = $BN"

if [[ "$BN" == "$HEXTIP" ]]; then
  echo "✅  SUCCESS – node booted at expected height $TIP"
  kill $NODEPID
  exit 0
else
  echo "❌  FAILED – expected $HEXTIP"
  kill $NODEPID
  exit 1
fi