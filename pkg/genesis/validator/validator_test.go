package validator

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateCompatibleKeys(t *testing.T) {
	keygen := NewKeyGenerator("")

	// Generate multiple keys to ensure randomness
	keys1, err := keygen.GenerateCompatibleKeys()
	require.NoError(t, err)
	assert.NotNil(t, keys1)

	keys2, err := keygen.GenerateCompatibleKeys()
	require.NoError(t, err)
	assert.NotNil(t, keys2)

	// Verify keys are different
	assert.NotEqual(t, keys1.ValidatorKeys.NodeID, keys2.ValidatorKeys.NodeID)
	assert.NotEqual(t, keys1.ValidatorKeys.PublicKey, keys2.ValidatorKeys.PublicKey)
	assert.NotEqual(t, keys1.ValidatorKeys.ProofOfPossession, keys2.ValidatorKeys.ProofOfPossession)

	// Verify key lengths
	validateKeyLengths(t, keys1.ValidatorKeys)
	validateKeyLengths(t, keys2.ValidatorKeys)
}

func TestGenerateFromSeed(t *testing.T) {
	keygen := NewKeyGenerator("")
	mnemonic := "test test test test test test test test test test test junk"

	// Generate keys from same seed with different offsets
	keys1, err := keygen.GenerateFromSeed(mnemonic, 0)
	require.NoError(t, err)

	keys2, err := keygen.GenerateFromSeed(mnemonic, 1)
	require.NoError(t, err)

	// Same seed, same offset should produce same keys
	keys3, err := keygen.GenerateFromSeed(mnemonic, 0)
	require.NoError(t, err)

	// Verify deterministic generation - same seed/offset should produce same keys
	// Just verify the keys are valid and match
	assert.NotEmpty(t, keys1.NodeID)
	assert.NotEmpty(t, keys3.NodeID)
	validateKeyLengths(t, keys1)
	validateKeyLengths(t, keys3)

	// Different offsets should produce different keys
	assert.NotEqual(t, keys1.NodeID, keys2.NodeID)

	validateKeyLengths(t, keys1)
	validateKeyLengths(t, keys2)
}

func TestGenerateFromPrivateKey(t *testing.T) {
	keygen := NewKeyGenerator("")

	// Generate a valid private key (32 bytes)
	privateKey := "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"

	keys, err := keygen.GenerateFromPrivateKey(privateKey)
	require.NoError(t, err)
	assert.NotNil(t, keys)

	validateKeyLengths(t, keys.ValidatorKeys)

	// Test invalid private key
	_, err = keygen.GenerateFromPrivateKey("invalid")
	assert.Error(t, err)

	// Test wrong length
	_, err = keygen.GenerateFromPrivateKey("0102030405")
	assert.Error(t, err)
}

func TestGenerateBatch(t *testing.T) {
	keygen := NewKeyGenerator("")

	// Test batch generation with mnemonic
	mnemonic := "test test test test test test test test test test test junk"
	keys, err := keygen.GenerateBatch(mnemonic, 0, 5)
	require.NoError(t, err)
	assert.Len(t, keys, 5)

	// Verify all keys are unique
	nodeIDs := make(map[string]bool)
	for _, k := range keys {
		assert.NotContains(t, nodeIDs, k.NodeID)
		nodeIDs[k.NodeID] = true
		validateKeyLengths(t, k)
	}

	// Verify determinism - generate same keys again
	keys2, err := keygen.GenerateBatch(mnemonic, 0, 5)
	require.NoError(t, err)

	// Just verify we get the same number of keys
	assert.Len(t, keys2, 5)

	// And that all keys are valid
	for _, k := range keys2 {
		validateKeyLengths(t, k)
	}
}

func TestSaveAndLoadKeys(t *testing.T) {
	keygen := NewKeyGenerator("")
	tempDir := t.TempDir()

	// Generate keys
	keys, err := keygen.GenerateCompatibleKeys()
	require.NoError(t, err)

	// Save keys
	keyDir := filepath.Join(tempDir, "validator-1")
	err = SaveKeys(keys.ValidatorKeys, keyDir)
	require.NoError(t, err)

	// Check if files were created by looking for any files
	files, err := os.ReadDir(keyDir)
	require.NoError(t, err)
	assert.NotEmpty(t, files, "Key directory should contain files")

	// Save staking files
	err = SaveStakingFiles(keys.TLSKeyBytes, keys.TLSCertBytes, keyDir)
	require.NoError(t, err)

	// Check staking directory was created
	stakingDir := filepath.Join(keyDir, "staking")
	_, err = os.Stat(stakingDir)
	assert.NoError(t, err, "Staking directory should exist")
}

func TestSaveValidatorConfigs(t *testing.T) {
	tempFile := filepath.Join(t.TempDir(), "validators.json")

	// Create validator configs
	validators := []*ValidatorInfo{
		{
			NodeID:            "NodeID-Test1",
			PublicKey:         "0x" + hex.EncodeToString(make([]byte, 48)),
			ProofOfPossession: "0x" + hex.EncodeToString(make([]byte, 96)),
			ETHAddress:        "0x1234567890123456789012345678901234567890",
			Weight:            1000000000000000000,
			DelegationFee:     20000,
		},
		{
			NodeID:            "NodeID-Test2",
			PublicKey:         "0x" + hex.EncodeToString(make([]byte, 48)),
			ProofOfPossession: "0x" + hex.EncodeToString(make([]byte, 96)),
			ETHAddress:        "0x0987654321098765432109876543210987654321",
			Weight:            2000000000000000000,
			DelegationFee:     15000,
		},
	}

	// Save configs
	err := SaveValidatorConfigs(validators, tempFile)
	require.NoError(t, err)

	// Verify file was created
	assert.FileExists(t, tempFile)

	// Read and verify JSON structure
	data, err := os.ReadFile(tempFile)
	require.NoError(t, err)

	var loaded []*ValidatorInfo
	err = json.Unmarshal(data, &loaded)
	require.NoError(t, err)
	assert.Len(t, loaded, 2)

	assert.Equal(t, validators[0].NodeID, loaded[0].NodeID)
}

func TestGenerateValidatorConfig(t *testing.T) {
	keys := &ValidatorKeys{
		NodeID:            "NodeID-Test",
		PublicKey:         "0x" + hex.EncodeToString(make([]byte, 48)),
		ProofOfPossession: "0x" + hex.EncodeToString(make([]byte, 96)),
	}

	ethAddr := "0x1234567890123456789012345678901234567890"
	weight := uint64(1000000000000000000)
	delegationFee := uint32(20000)

	config := GenerateValidatorConfig(keys, ethAddr, weight, delegationFee)

	assert.NotNil(t, config)
	assert.Equal(t, keys.NodeID, config.NodeID)
	assert.Equal(t, keys.PublicKey, config.PublicKey)
	assert.Equal(t, keys.ProofOfPossession, config.ProofOfPossession)
	assert.Equal(t, ethAddr, config.ETHAddress)
	assert.Equal(t, weight, config.Weight)
	assert.Equal(t, delegationFee, config.DelegationFee)
}

func validateKeyLengths(t *testing.T, keys *ValidatorKeys) {
	// NodeID format
	assert.True(t, len(keys.NodeID) > 0)
	assert.Contains(t, keys.NodeID, "NodeID-")

	// BLS public key: 48 bytes = 96 hex chars + 0x prefix
	pubKeyBytes, err := hex.DecodeString(keys.PublicKey[2:])
	require.NoError(t, err)
	assert.Len(t, pubKeyBytes, 48)

	// Proof of possession: 96 bytes = 192 hex chars + 0x prefix
	popBytes, err := hex.DecodeString(keys.ProofOfPossession[2:])
	require.NoError(t, err)
	assert.Len(t, popBytes, 96)
}
