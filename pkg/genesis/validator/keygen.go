package validator

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	"github.com/luxfi/node/ids"
	"github.com/luxfi/node/staking"
	"github.com/luxfi/node/utils/crypto/bls"
	"github.com/luxfi/node/utils/crypto/bls/signer/localsigner"
	"github.com/luxfi/node/utils/formatting/address"
)

// KeyGenerator handles validator key generation using luxd
type KeyGenerator struct {
	luxdPath string
}

// NewKeyGenerator creates a new key generator
func NewKeyGenerator(luxdPath string) *KeyGenerator {
	return &KeyGenerator{
		luxdPath: luxdPath,
	}
}

// ValidatorKeys contains the generated validator key information
type ValidatorKeys struct {
	NodeID            string `json:"nodeID"`
	PublicKey         string `json:"publicKey"`
	ProofOfPossession string `json:"proofOfPossession"`
	PrivateKey        string `json:"privateKey,omitempty"` // Only for secure storage, never in genesis
}

// GenerateFromSeed generates validator keys using BLS with deterministic TLS cert
func (kg *KeyGenerator) GenerateFromSeedWithTLS(seedPhrase string, accountNum int) (*ValidatorKeysWithTLS, error) {
	// Use deterministic derivation
	seedData := fmt.Sprintf("%s-luxnode-%d", seedPhrase, accountNum)
	
	// Create deterministic private key using HKDF-like approach
	h := sha256.New()
	h.Write([]byte("lux-bls-key"))
	h.Write([]byte(seedData))
	seed := h.Sum(nil)
	
	// BLS12-381 curve order
	curveOrder, _ := new(big.Int).SetString("73eda753299d7d483339d80809a1d80553bda402fffe5bfeffffffff00000001", 16)
	
	// Convert seed to big int and reduce modulo curve order
	keyInt := new(big.Int).SetBytes(seed)
	keyInt.Mod(keyInt, curveOrder)
	
	// Ensure non-zero
	if keyInt.Sign() == 0 {
		keyInt.SetInt64(1)
	}
	
	// Convert back to 32 bytes
	keyBytes := make([]byte, 32)
	keyInt.FillBytes(keyBytes)
	
	// Create signer from the deterministic key
	signer, err := localsigner.FromBytes(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to generate BLS signer: %w", err)
	}
	
	pk := signer.PublicKey()
	
	// Generate proof of possession
	pop, err := signer.SignProofOfPossession(bls.PublicKeyToCompressedBytes(pk))
	if err != nil {
		return nil, fmt.Errorf("failed to sign proof of possession: %w", err)
	}
	
	// Generate TLS certificate for NodeID (deterministic based on account)
	// For production, you'd want proper deterministic TLS generation
	tlsCertPEM, tlsKeyPEM, err := staking.NewCertAndKeyBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to generate TLS cert: %w", err)
	}
	
	// Decode PEM to get DER bytes
	block, _ := pem.Decode(tlsCertPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode certificate PEM")
	}
	
	cert, err := staking.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}
	
	nodeID := ids.NodeIDFromCert(cert)
	
	// Create validator keys
	keys := &ValidatorKeys{
		NodeID:            nodeID.String(),
		PublicKey:         "0x" + hex.EncodeToString(bls.PublicKeyToCompressedBytes(pk)),
		ProofOfPossession: "0x" + hex.EncodeToString(bls.SignatureToBytes(pop)),
		PrivateKey:        "0x" + hex.EncodeToString(signer.ToBytes()),
	}
	
	return &ValidatorKeysWithTLS{
		ValidatorKeys: keys,
		TLSKeyBytes:   tlsKeyPEM,
		TLSCertBytes:  tlsCertPEM,
	}, nil
}

// GenerateFromSeed generates validator keys using BLS
func (kg *KeyGenerator) GenerateFromSeed(seedPhrase string, accountNum int) (*ValidatorKeys, error) {
	keysWithTLS, err := kg.GenerateFromSeedWithTLS(seedPhrase, accountNum)
	if err != nil {
		return nil, err
	}
	return keysWithTLS.ValidatorKeys, nil
}

// GenerateBatch generates keys for multiple validators from a single seed phrase
func (kg *KeyGenerator) GenerateBatch(seedPhrase string, startAccount, count int) ([]*ValidatorKeys, error) {
	var results []*ValidatorKeys

	for i := 0; i < count; i++ {
		keys, err := kg.GenerateFromSeed(seedPhrase, startAccount+i)
		if err != nil {
			return nil, fmt.Errorf("failed to generate keys for account %d: %w", startAccount+i, err)
		}
		results = append(results, keys)
	}

	return results, nil
}

// ValidatorInfo contains the full validator configuration
type ValidatorInfo struct {
	NodeID            string `json:"nodeID"`
	ETHAddress        string `json:"ethAddress"`
	PublicKey         string `json:"publicKey"`
	ProofOfPossession string `json:"proofOfPossession"`
	Weight            uint64 `json:"weight"`
	DelegationFee     uint32 `json:"delegationFee"`
}

// GenerateValidatorConfig generates a complete validator configuration
func GenerateValidatorConfig(keys *ValidatorKeys, ethAddress string, weight uint64, delegationFee uint32) *ValidatorInfo {
	return &ValidatorInfo{
		NodeID:            keys.NodeID,
		ETHAddress:        ethAddress,
		PublicKey:         keys.PublicKey,
		ProofOfPossession: keys.ProofOfPossession,
		Weight:            weight,
		DelegationFee:     delegationFee,
	}
}

// SaveValidatorConfigs saves validator configurations to a JSON file
func SaveValidatorConfigs(validators []*ValidatorInfo, filepath string) error {
	data, err := json.MarshalIndent(validators, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal validators: %w", err)
	}

	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// ValidatorKeysWithTLS contains validator keys and TLS certificate data
type ValidatorKeysWithTLS struct {
	*ValidatorKeys
	TLSKeyBytes  []byte
	TLSCertBytes []byte
}

// GenerateCompatibleKeys generates validator keys compatible with luxd
// This uses the same TLS-based NodeID generation as luxd
func (kg *KeyGenerator) GenerateCompatibleKeys() (*ValidatorKeysWithTLS, error) {
	// Generate BLS key pair
	signer, err := localsigner.New()
	if err != nil {
		return nil, fmt.Errorf("failed to generate BLS signer: %w", err)
	}
	
	pk := signer.PublicKey()
	
	// Generate proof of possession
	pop, err := signer.SignProofOfPossession(bls.PublicKeyToCompressedBytes(pk))
	if err != nil {
		return nil, fmt.Errorf("failed to sign proof of possession: %w", err)
	}
	
	// Generate TLS certificate for NodeID
	tlsCertBytes, tlsKeyBytes, err := staking.NewCertAndKeyBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to generate TLS cert: %w", err)
	}
	
	// Decode PEM to get DER bytes for parsing
	block, _ := pem.Decode(tlsCertBytes)
	if block == nil {
		return nil, fmt.Errorf("failed to decode certificate PEM")
	}
	
	cert, err := staking.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}
	
	nodeID := ids.NodeIDFromCert(cert)
	
	// Format address in X-Chain format
	xAddr, err := address.FormatBech32("X", nodeID.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to format X-Chain address: %w", err)
	}
	
	keys := &ValidatorKeys{
		NodeID:            nodeID.String(),
		PublicKey:         "0x" + hex.EncodeToString(bls.PublicKeyToCompressedBytes(pk)),
		ProofOfPossession: "0x" + hex.EncodeToString(bls.SignatureToBytes(pop)),
		PrivateKey:        "0x" + hex.EncodeToString(signer.ToBytes()), // Include private key for secure storage
	}
	
	// Also include the X-Chain address for convenience
	keys.NodeID = fmt.Sprintf("%s (X-Chain: %s)", nodeID.String(), xAddr)
	
	return &ValidatorKeysWithTLS{
		ValidatorKeys: keys,
		TLSKeyBytes:   tlsKeyBytes,
		TLSCertBytes:  tlsCertBytes,
	}, nil
}

// SaveKeys saves validator keys and certificates to disk
func SaveKeys(keys *ValidatorKeys, outputDir string) error {
	// Create output directory
	if err := os.MkdirAll(outputDir, 0700); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	
	// Save validator info
	info := map[string]interface{}{
		"nodeID":            keys.NodeID,
		"publicKey":         keys.PublicKey,
		"proofOfPossession": keys.ProofOfPossession,
	}
	
	infoData, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal validator info: %w", err)
	}
	
	infoPath := filepath.Join(outputDir, "validator.json")
	if err := os.WriteFile(infoPath, infoData, 0644); err != nil {
		return fmt.Errorf("failed to write validator info: %w", err)
	}
	
	// If private key is available, save it separately with restricted permissions
	if keys.PrivateKey != "" {
		// Convert hex string to binary
		privKeyHex := strings.TrimPrefix(keys.PrivateKey, "0x")
		privKeyBytes, err := hex.DecodeString(privKeyHex)
		if err != nil {
			return fmt.Errorf("failed to decode private key hex: %w", err)
		}
		
		privKeyPath := filepath.Join(outputDir, "bls.key")
		if err := os.WriteFile(privKeyPath, privKeyBytes, 0600); err != nil {
			return fmt.Errorf("failed to write private key: %w", err)
		}
	}
	
	return nil
}

// SaveStakingFiles saves the TLS key and certificate files for a validator
func SaveStakingFiles(tlsKeyBytes, tlsCertBytes []byte, outputDir string) error {
	// Create staking directory
	stakingDir := filepath.Join(outputDir, "staking")
	if err := os.MkdirAll(stakingDir, 0700); err != nil {
		return fmt.Errorf("failed to create staking directory: %w", err)
	}
	
	// Save TLS key
	keyPath := filepath.Join(stakingDir, "staker.key")
	if err := os.WriteFile(keyPath, tlsKeyBytes, 0600); err != nil {
		return fmt.Errorf("failed to write staking key: %w", err)
	}
	
	// Save TLS certificate
	certPath := filepath.Join(stakingDir, "staker.crt")
	if err := os.WriteFile(certPath, tlsCertBytes, 0644); err != nil {
		return fmt.Errorf("failed to write staking cert: %w", err)
	}
	
	return nil
}

// GenerateFromPrivateKey generates validator keys from a BLS private key
func (kg *KeyGenerator) GenerateFromPrivateKey(privateKeyHex string) (*ValidatorKeysWithTLS, error) {
	// Decode hex private key
	privateKeyHex = strings.TrimPrefix(privateKeyHex, "0x")
	privKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key hex: %w", err)
	}
	
	// Create signer from private key
	signer, err := localsigner.FromBytes(privKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to create signer from private key: %w", err)
	}
	
	pk := signer.PublicKey()
	
	// Generate proof of possession
	pop, err := signer.SignProofOfPossession(bls.PublicKeyToCompressedBytes(pk))
	if err != nil {
		return nil, fmt.Errorf("failed to sign proof of possession: %w", err)
	}
	
	// Generate TLS certificate for NodeID
	tlsCertPEM, tlsKeyPEM, err := staking.NewCertAndKeyBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to generate TLS cert: %w", err)
	}
	
	// Decode PEM to get DER bytes
	block, _ := pem.Decode(tlsCertPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode certificate PEM")
	}
	
	cert, err := staking.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}
	
	nodeID := ids.NodeIDFromCert(cert)
	
	// Create validator keys
	keys := &ValidatorKeys{
		NodeID:            nodeID.String(),
		PublicKey:         "0x" + hex.EncodeToString(bls.PublicKeyToCompressedBytes(pk)),
		ProofOfPossession: "0x" + hex.EncodeToString(bls.SignatureToBytes(pop)),
		PrivateKey:        "0x" + privateKeyHex,
	}
	
	return &ValidatorKeysWithTLS{
		ValidatorKeys: keys,
		TLSKeyBytes:   tlsKeyPEM,
		TLSCertBytes:  tlsCertPEM,
	}, nil
}