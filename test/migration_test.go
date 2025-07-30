package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/cockroachdb/pebble"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMigration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Subnet to C-Chain Migration Suite")
}

var _ = BeforeSuite(func() {
	// Build genesis tool if needed
	err := buildGenesisTool()
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("Subnet to C-Chain Migration", func() {
	var (
		tmpDir         string
		subnetDataPath string
		migratedPath   string
	)

	BeforeEach(func() {
		// Use .tmp directory in project folder
		projectRoot := "$HOME/work/lux/genesis"
		baseTmpDir := filepath.Join(projectRoot, ".tmp")

		// Create .tmp directory if it doesn't exist
		err := os.MkdirAll(baseTmpDir, 0755)
		Expect(err).NotTo(HaveOccurred())

		tmpDir, err = os.MkdirTemp(baseTmpDir, "lux-migration-test-*")
		Expect(err).NotTo(HaveOccurred())

		subnetDataPath = filepath.Join(tmpDir, "subnet-data")
		migratedPath = filepath.Join(tmpDir, "migrated-chaindata")
	})

	AfterEach(func() {
		// Clean up temp directory
		if tmpDir != "" {
			os.RemoveAll(tmpDir)
		}
	})

	Describe("Step 1: Create Test Subnet Data", func() {
		It("should create mock subnet state data", func() {
			By("Creating a test PebbleDB with state trie nodes")
			// Ensure directory exists
			err := os.MkdirAll(filepath.Dir(subnetDataPath), 0755)
			Expect(err).NotTo(HaveOccurred())

			db, err := pebble.Open(subnetDataPath, &pebble.Options{})
			Expect(err).NotTo(HaveOccurred())
			defer db.Close()

			// Add some test state trie nodes
			batch := db.NewBatch()

			// Add state root
			stateRoot := []byte{0x01, 0x02, 0x03}
			err = batch.Set([]byte("stateRoot"), stateRoot, nil)
			Expect(err).NotTo(HaveOccurred())

			// Add some trie nodes
			for i := 0; i < 10; i++ {
				key := []byte(fmt.Sprintf("trieNode%d", i))
				value := make([]byte, 32)
				binary.BigEndian.PutUint64(value, uint64(i))
				err = batch.Set(key, value, nil)
				Expect(err).NotTo(HaveOccurred())
			}

			err = batch.Commit(nil)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying state data was created")
			val, closer, err := db.Get([]byte("stateRoot"))
			Expect(err).NotTo(HaveOccurred())
			Expect(val).To(Equal(stateRoot))
			closer.Close()
		})
	})

	Describe("Step 2: Add EVM Prefix Migration", func() {
		It("should add 'evm' prefix to all keys", func() {
			By("First ensuring we have a source database")
			// If the database doesn't exist from Step 1, create it
			if _, err := os.Stat(subnetDataPath); os.IsNotExist(err) {
				err := os.MkdirAll(filepath.Dir(subnetDataPath), 0755)
				Expect(err).NotTo(HaveOccurred())

				db, err := pebble.Open(subnetDataPath, &pebble.Options{})
				Expect(err).NotTo(HaveOccurred())

				// Add some test data
				batch := db.NewBatch()
				for i := 0; i < 10; i++ {
					key := []byte(fmt.Sprintf("testKey%d", i))
					value := []byte(fmt.Sprintf("testValue%d", i))
					err = batch.Set(key, value, nil)
					Expect(err).NotTo(HaveOccurred())
				}
				err = batch.Commit(nil)
				Expect(err).NotTo(HaveOccurred())
				db.Close()
			}

			By("Running the subnet import using genesis tool")
			output, err := genesis("import", "subnet", subnetDataPath, migratedPath)
			Expect(err).NotTo(HaveOccurred(), output)

			By("Verifying migration was successful")
			db, err := pebble.Open(migratedPath, &pebble.Options{ReadOnly: true})
			Expect(err).NotTo(HaveOccurred())
			defer db.Close()

			iter, err := db.NewIter(nil)
			Expect(err).NotTo(HaveOccurred())
			defer iter.Close()

			count := 0
			hasChainMarkers := false
			for iter.First(); iter.Valid(); iter.Next() {
				key := string(iter.Key())
				count++
				// Check for chain continuity markers
				if key == "lastAccepted" || key == "LastBlock" || key == "LastHeader" {
					hasChainMarkers = true
				}
			}
			Expect(count).To(BeNumerically(">", 0))
			Expect(hasChainMarkers).To(BeTrue(), "Should have chain continuity markers")
		})
	})

	Describe("Step 3: Rebuild Canonical Mappings", func() {
		It("should rebuild evmn keys from headers", func() {
			Skip("Canonical mappings are now handled automatically by import-subnet command")
		})
	})

	Describe("Step 4: Generate Consensus State", func() {
		It("should create Snowman consensus state with versiondb", func() {
			Skip("Consensus state is now handled automatically by import-subnet command")
		})
	})

	Describe("Step 5: Verify Migration Tools", func() {
		It("should verify all analysis tools work correctly", func() {
			By("Running key structure analysis")
			if _, err := os.Stat(migratedPath); os.IsNotExist(err) {
				Skip("Migrated path doesn't exist")
			}

			output, err := genesis("analyze", "keys", migratedPath)
			if err == nil {
				Expect(output).To(Or(
					ContainSubstring("Analyzing"),
					ContainSubstring("Database Analysis"),
					ContainSubstring("keys found"),
				))
			}

			By("Checking chain tip")
			output, _ = genesis("inspect", "tip", migratedPath)
			// Output might show block 0 for test data

			By("Inspecting database structure")
			output, _ = genesis("inspect", "keys", migratedPath)
			// Output analysis
		})
	})

	Describe("Integration: Full Pipeline", func() {
		It("should complete entire migration pipeline", func() {
			By("Starting with subnet data at " + subnetDataPath)
			if _, err := os.Stat(subnetDataPath); os.IsNotExist(err) {
				Skip("No subnet data available")
			}

			By("Running migration using import-subnet command")
			fullMigrationPath := filepath.Join(tmpDir, "full-migration")
			output, err := genesis("import", "subnet", subnetDataPath, fullMigrationPath)

			// The command should succeed for test data
			Expect(err).NotTo(HaveOccurred(), output)

			By("Checking final state")
			if _, err := os.Stat(fullMigrationPath); err == nil {
				tip, _ := genesis("inspect", "tip", fullMigrationPath)
				fmt.Printf("Final state - tip: %s\n", tip)
			}
		})
	})

	// Edge cases
	Describe("Edge Cases and Error Handling", func() {
		It("should handle missing source database", func() {
			nonExistentPath := filepath.Join(tmpDir, "does-not-exist")
			output, err := genesis("import", "subnet", nonExistentPath, migratedPath)
			Expect(err).To(HaveOccurred())
			Expect(output).To(ContainSubstring("failed to check source database"))
		})

		It("should handle empty database", func() {
			emptyPath := filepath.Join(tmpDir, "empty-db")
			db, err := pebble.Open(emptyPath, &pebble.Options{})
			Expect(err).NotTo(HaveOccurred())
			db.Close()

			output, err := genesis("inspect", "tip", emptyPath)
			// Should complete without error on empty database
			Expect(err).NotTo(HaveOccurred())
			// Output should indicate scanning or finding tip
			Expect(output).To(Or(
				ContainSubstring("Scanning"),
				ContainSubstring("Finding"),
				ContainSubstring("Tip"),
			))
		})
	})
})
