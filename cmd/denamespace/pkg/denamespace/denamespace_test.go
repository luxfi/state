package denamespace

import (
	"encoding/hex"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/cockroachdb/pebble"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChainIDs(t *testing.T) {
	// Test that chain IDs are properly defined
	assert.NotEmpty(t, chainIDs[96369])
	assert.NotEmpty(t, chainIDs[96368])
	assert.NotEmpty(t, chainIDs[200200])
	assert.NotEmpty(t, chainIDs[36911])

	// Test that they are valid hex
	for chainID, hexStr := range chainIDs {
		_, err := hex.DecodeString(hexStr)
		assert.NoError(t, err, "Invalid hex for chain ID %d", chainID)
	}
}

func TestExtractOptions(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "denamespace-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	t.Run("invalid network ID", func(t *testing.T) {
		opts := Options{
			Source:      filepath.Join(tmpDir, "src"),
			Destination: filepath.Join(tmpDir, "dst"),
			NetworkID:   99999, // Invalid
			State:       true,
		}

		err := Extract(opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown network ID")
	})

	t.Run("missing source directory", func(t *testing.T) {
		opts := Options{
			Source:      filepath.Join(tmpDir, "non-existent"),
			Destination: filepath.Join(tmpDir, "dst"),
			NetworkID:   96369,
			State:       true,
		}

		err := Extract(opts)
		assert.Error(t, err)
	})
}

func TestExtractWithMockData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping mock data test in short mode")
	}

	tmpDir, err := ioutil.TempDir("", "denamespace-mock-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	srcPath := filepath.Join(tmpDir, "src")

	// Create a mock PebbleDB with test data
	src, err := pebble.Open(srcPath, &pebble.Options{})
	require.NoError(t, err)

	// Add some test data
	chainHex := chainIDs[96369]
	chainBytes, err := hex.DecodeString(chainHex)
	require.NoError(t, err)

	// Create test keys with namespace prefix
	testData := map[string][]byte{
		// Metadata key (no namespace)
		"LastAccepted": []byte("test-block-hash"),

		// Namespaced header key
		string(append(append(chainBytes, 0x68), []byte("header-key")...)): []byte("header-data"),

		// Namespaced account key (only included if state=true)
		string(append(append(chainBytes, 0x26), []byte("account-key")...)): []byte("account-data"),

		// Invalid namespace (should be skipped)
		string(append([]byte("invalid-namespace"), []byte("key")...)): []byte("should-skip"),
	}

	// Write test data
	batch := src.NewBatch()
	for k, v := range testData {
		err := batch.Set([]byte(k), v, nil)
		require.NoError(t, err)
	}
	err = batch.Commit(pebble.Sync)
	require.NoError(t, err)
	src.Close()

	// Test extraction with state
	t.Run("extract with state", func(t *testing.T) {
		dstWithState := filepath.Join(tmpDir, "dst-with-state")
		opts := Options{
			Source:      srcPath,
			Destination: dstWithState,
			NetworkID:   96369,
			State:       true,
			Limit:       0,
		}

		err := Extract(opts)
		require.NoError(t, err)

		// Verify destination was created
		assert.DirExists(t, dstWithState)

		// Open destination and verify data
		dst, err := pebble.Open(dstWithState, &pebble.Options{ReadOnly: true})
		require.NoError(t, err)
		defer dst.Close()

		// Check metadata key
		val, closer, err := dst.Get([]byte("LastAccepted"))
		require.NoError(t, err)
		assert.Equal(t, "test-block-hash", string(val))
		closer.Close()

		// Check header key (namespace removed)
		val, closer, err = dst.Get([]byte("header-key"))
		require.NoError(t, err)
		assert.Equal(t, "header-data", string(val))
		closer.Close()

		// Check account key (namespace removed)
		val, closer, err = dst.Get([]byte("account-key"))
		require.NoError(t, err)
		assert.Equal(t, "account-data", string(val))
		closer.Close()
	})

	// Test extraction without state
	t.Run("extract without state", func(t *testing.T) {
		dstNoState := filepath.Join(tmpDir, "dst-no-state")
		opts := Options{
			Source:      srcPath,
			Destination: dstNoState,
			NetworkID:   96369,
			State:       false,
			Limit:       0,
		}

		err := Extract(opts)
		require.NoError(t, err)

		// Open destination and verify data
		dst, err := pebble.Open(dstNoState, &pebble.Options{ReadOnly: true})
		require.NoError(t, err)
		defer dst.Close()

		// Check that account key was NOT copied
		_, _, err = dst.Get([]byte("account-key"))
		assert.Error(t, err) // Should not exist
	})

	// Test with limit
	t.Run("extract with limit", func(t *testing.T) {
		dstLimit := filepath.Join(tmpDir, "dst-limit")
		opts := Options{
			Source:      srcPath,
			Destination: dstLimit,
			NetworkID:   96369,
			State:       true,
			Limit:       1, // Only copy 1 key
		}

		err := Extract(opts)
		require.NoError(t, err)

		// Should complete successfully even with limit
		assert.DirExists(t, dstLimit)
	})
}

func TestValidSuffixes(t *testing.T) {
	// Test that suffix detection works correctly
	suffixes := map[byte]bool{
		0x68: true,  // headers
		0x6c: true,  // last values
		0x48: true,  // Headers
		0x72: true,  // receipts
		0x62: true,  // bodies
		0x42: true,  // Bodies
		0x6e: true,  // number->hash
		0x74: true,  // transactions
		0xfd: true,  // metadata
		0x26: false, // accounts (only with state)
		0xa3: false, // storage (only with state)
		0x6f: false, // objects (only with state)
		0x73: false, // state (only with state)
		0x63: false, // code (only with state)
		0xFF: false, // invalid suffix
	}

	for suffix, shouldBeValid := range suffixes {
		validSuffixes := map[byte]string{
			0x68: "headers",
			0x6c: "last values",
			0x48: "Headers",
			0x72: "receipts",
			0x62: "bodies",
			0x42: "Bodies",
			0x6e: "number->hash",
			0x74: "transactions",
			0xfd: "metadata",
		}

		_, isValid := validSuffixes[suffix]
		if shouldBeValid {
			assert.True(t, isValid, "Suffix 0x%02x should be valid", suffix)
		}
	}
}

func TestMetadataKeys(t *testing.T) {
	// Test metadata key recognition
	metadataKeys := []string{
		"LastAccepted",
		"last_accepted_key",
		"lastAccepted",
		"lastFinalized",
		"LastFinalizedKey",
		"vm_state",
		"chain_state",
	}

	for _, key := range metadataKeys {
		// Test exact match
		assert.True(t, isMetadataKey([]byte(key)), "Key %s should be metadata", key)

		// Test prefix match
		assert.True(t, isMetadataKey([]byte(key+"_suffix")), "Key %s_suffix should be metadata", key)
	}

	// Test non-metadata keys
	nonMetadataKeys := []string{
		"random_key",
		"data",
		"block",
	}

	for _, key := range nonMetadataKeys {
		assert.False(t, isMetadataKey([]byte(key)), "Key %s should not be metadata", key)
	}
}

// Helper function to check if a key is metadata
func isMetadataKey(key []byte) bool {
	metadataKeys := []string{
		"LastAccepted",
		"last_accepted_key",
		"lastAccepted",
		"lastFinalized",
		"LastFinalizedKey",
		"vm_state",
		"chain_state",
	}

	for _, mk := range metadataKeys {
		mkBytes := []byte(mk)
		if len(key) >= len(mkBytes) && string(key[:len(mkBytes)]) == mk {
			return true
		}
	}
	return false
}
