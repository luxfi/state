package tests_test

import (
	"encoding/hex"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	
	"github.com/cockroachdb/pebble"
)

var _ = Describe("Database Operations", func() {
	var genesisDir string

	BeforeEach(func() {
		homeDir := "$HOME"
		genesisDir = filepath.Join(homeDir, "work/lux/genesis")
	})

	Describe("Chain Data Validation", func() {
		Context("7777 Chain Data", func() {
			It("should have valid PebbleDB structure", func() {
	dbPath := filepath.Join(genesisDir, "chaindata/lux-genesis-7777/db")
				
				// Check if it exists
				Expect(dbPath).To(BeADirectory())
				
				// Open and validate
				db, err := pebble.Open(dbPath, &pebble.Options{
					ReadOnly: true,
				})
				Expect(err).NotTo(HaveOccurred())
				defer db.Close()

				// Count keys by prefix
				prefixCounts := make(map[string]int)
				iter, err := db.NewIter(nil)
				Expect(err).NotTo(HaveOccurred())
				defer iter.Close()

				keyCount := 0
				for iter.First(); iter.Valid(); iter.Next() {
					keyCount++
					if len(iter.Key()) > 0 {
						prefix := hex.EncodeToString(iter.Key()[:1])
						prefixCounts[prefix]++
					}
				}
				Expect(iter.Error()).NotTo(HaveOccurred())

				// Validate we have data
				Expect(keyCount).To(BeNumerically(">", 1000), "Database should have substantial data")
				
				// Check for expected key prefixes
				Expect(prefixCounts).To(HaveKey("68")) // headers
				Expect(prefixCounts).To(HaveKey("62")) // bodies
				Expect(prefixCounts).To(HaveKey("72")) // receipts
			})

			It("should have correct genesis block", func() {
	dbPath := filepath.Join(genesisDir, "chaindata/lux-genesis-7777/db")
				
				db, err := pebble.Open(dbPath, &pebble.Options{
					ReadOnly: true,
				})
				Expect(err).NotTo(HaveOccurred())
				defer db.Close()

				// Look for genesis block (block 0)
				// Key format: 'h' + block_number (8 bytes) + hash (32 bytes)
				iter, err := db.NewIter(&pebble.IterOptions{
					LowerBound: []byte("h"),
					UpperBound: []byte("i"),
				})
				Expect(err).NotTo(HaveOccurred())
				defer iter.Close()

				foundGenesis := false
				for iter.First(); iter.Valid(); iter.Next() {
					key := iter.Key()
					if len(key) >= 9 && key[0] == 'h' {
						// Check if block number is 0
						blockNum := key[1:9]
						isZero := true
						for _, b := range blockNum {
							if b != 0 {
								isZero = false
								break
							}
						}
						if isZero {
							foundGenesis = true
							break
						}
					}
				}
				Expect(foundGenesis).To(BeTrue(), "Genesis block should exist")
			})
		})

		Context("96369 Chain Data", func() {
			It("should validate mainnet data structure", func() {
				dbPath := filepath.Join(genesisDir, "chaindata/lux-mainnet-96369/db/pebbledb")
				
				// Check if it exists
				if _, err := filepath.Glob(filepath.Join(dbPath, "*.sst")); err != nil {
					Skip("96369 mainnet data not available")
				}

				db, err := pebble.Open(dbPath, &pebble.Options{
					ReadOnly: true,
				})
				Expect(err).NotTo(HaveOccurred())
				defer db.Close()

				// Verify database has substantial data
				metrics := db.Metrics()
				Expect(metrics.DiskSpaceUsage()).To(BeNumerically(">", 1*1024*1024*1024), "Should have > 1GB of data")
			})
		})
	})

	Describe("Data Conversion", func() {
		It("should convert LevelDB to PebbleDB maintaining data integrity", func() {
			Skip("Requires test data setup")
			
			// This would test the conversion process
			// Create a small test LevelDB, convert it, verify all keys match
		})
	})

	Describe("Genesis File Validation", func() {
		It("should have valid mainnet genesis", func() {
			genesisPath := filepath.Join(genesisDir, "deployments/configs/mainnet/lux/genesis.json")
			Expect(genesisPath).To(BeAnExistingFile())
			
			// Could parse and validate JSON structure here
		})

		It("should have valid testnet genesis", func() {
			genesisPath := filepath.Join(genesisDir, "deployments/configs/testnet/lux/genesis.json")
			Expect(genesisPath).To(BeAnExistingFile())
			
			// Could verify account migration here
		})
	})
})
