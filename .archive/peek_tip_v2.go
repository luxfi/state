package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"

	"github.com/ava-labs/avalanchego/database/prefixdb"
	"github.com/cockroachdb/pebble"
)

func main() {
	dbPath := flag.String("db", "", "path to *evm* PebbleDB")
	flag.Parse()
	if *dbPath == "" {
		log.Fatal("--db is required")
	}

	base, err := pebble.Open(*dbPath, &pebble.Options{})
	if err != nil {
		log.Fatalf("open: %v", err)
	}
	defer base.Close()

	// Wrap with the same namespace Coreth uses
	evm := prefixdb.New([]byte("evm"), base)

	it := evm.NewIter(nil)
	defer it.Close()

	prefix := []byte{'n'} // evmn --> after prefixdb, first byte is 'n'
	var tip uint64
	for it.SeekGE(prefix); it.Valid() && it.Key()[0] == 'n'; it.Next() {
		h := binary.BigEndian.Uint64(it.Key()[1:9]) // 'n' + 8‑byte height
		if h > tip {
			tip = h
		}
	}
	if tip == 0 {
		log.Fatal("no evmn keys found")
	}
	fmt.Print(tip)
}
