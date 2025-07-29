package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"time"
	
	"github.com/cockroachdb/pebble"
)

func main() {
	// Open consensus DB
	consDB, err := pebble.Open("runtime/luxd-final/db/chains/X6CU5qgMJfzsTB9UWxj2ZY5hd68x41HfZ4m4hCBWbHuj1Ebc3/db", &pebble.Options{})
	if err != nil {
		log.Fatal(err)
	}
	defer consDB.Close()
	
	// Get timestamp
	if val, closer, err := consDB.Get([]byte("timestamp")); err == nil && len(val) == 8 {
		timestamp := binary.BigEndian.Uint64(val)
		fmt.Printf("Timestamp: %d (Unix seconds)\n", timestamp)
		fmt.Printf("Human readable: %s\n", time.Unix(int64(timestamp), 0).Format(time.RFC3339))
		closer.Close()
	} else {
		fmt.Println("Timestamp not found in consensus DB")
	}
}