package main

import (
	"fmt"
	"os"
)

func main() {
	// This patch creates a special initialization path for migrated data
	// that completely bypasses normal genesis initialization
	
	patch := `--- a/node/vms/cchainvm/vm.go
+++ b/node/vms/cchainvm/vm.go
@@ -85,6 +85,27 @@ func (vm *VM) Initialize(
 	vm.shutdownChan = make(chan struct{})
 	vm.builtBlocks = make(map[ids.ID]*Block)
 	
+	// MIGRATION DETECTION: Check if we have migrated data BEFORE any initialization
+	// We need to check at the C-Chain database level, not the wrapped level
+	hasMigratedData := false
+	migratedHeight := uint64(0)
+	
+	// The database passed to us is already prefixed with chain ID
+	// We need to check for the EVM database inside it
+	tempDB := WrapDatabase(db)
+	if heightBytes, err := tempDB.Get([]byte("Height")); err == nil && len(heightBytes) == 8 {
+		height := binary.BigEndian.Uint64(heightBytes)
+		if height > 0 {
+			hasMigratedData = true
+			migratedHeight = height
+			fmt.Printf("DETECTED MIGRATED DATA AT HEIGHT %d\\n", height)
+			
+			// Log to Avalanche logger too
+			vm.ctx.Log.Info("Detected migrated blockchain data",
+				zap.Uint64("height", height),
+			)
+		}
+	}
+	
 	// Create a database wrapper first to check for migrated data
 	vm.ethDB = WrapDatabase(db)
 	
@@ -168,6 +189,15 @@ func (vm *VM) Initialize(
 	// Create minimal Ethereum backend
 	var err error
 	if hasMigratedData {
+		// CRITICAL: Skip all genesis processing for migrated data
+		fmt.Printf("MIGRATION MODE ACTIVE: Loading blockchain from height %d\\n", migratedHeight)
+		
+		// Create a special backend that doesn't touch genesis
+		vm.backend, err = NewMigratedBackend(vm.ethDB, migratedHeight)
+		if err != nil {
+			return fmt.Errorf("failed to create migrated backend: %w", err)
+		}
+	} else {
 		fmt.Printf("Creating backend for migrated data at height %d\\n", migratedHeight)
 		vm.backend, err = NewMinimalEthBackendForMigration(vm.ethDB, &vm.ethConfig, nil, migratedHeight)
 	} else {
--- a/node/vms/cchainvm/backend.go
+++ b/node/vms/cchainvm/backend.go
@@ -36,6 +36,89 @@ type MinimalEthBackend struct {
 	networkID   uint64
 }
 
+// NewMigratedBackend creates a special backend for fully migrated data
+// This completely bypasses genesis initialization
+func NewMigratedBackend(db ethdb.Database, migratedHeight uint64) (*MinimalEthBackend, error) {
+	fmt.Printf("Creating migrated backend for height %d\\n", migratedHeight)
+	
+	// Create a minimal chain config
+	chainConfig := &params.ChainConfig{
+		ChainID:                 big.NewInt(96369),
+		HomesteadBlock:          big.NewInt(0),
+		EIP150Block:             big.NewInt(0),
+		EIP155Block:             big.NewInt(0),
+		EIP158Block:             big.NewInt(0),
+		ByzantiumBlock:          big.NewInt(0),
+		ConstantinopleBlock:     big.NewInt(0),
+		PetersburgBlock:         big.NewInt(0),
+		IstanbulBlock:           big.NewInt(0),
+		BerlinBlock:             big.NewInt(0),
+		LondonBlock:             big.NewInt(0),
+		TerminalTotalDifficulty: common.Big0,
+	}
+	
+	// Create a dummy consensus engine
+	engine := &dummyEngine{}
+	
+	// Read the head hash from migrated data
+	blockNumBytes := make([]byte, 8)
+	binary.BigEndian.PutUint64(blockNumBytes, migratedHeight)
+	canonicalKey := append([]byte{0x68}, blockNumBytes...)
+	canonicalKey = append(canonicalKey, 0x6e)
+	
+	var headHash common.Hash
+	if val, err := db.Get(canonicalKey); err == nil && len(val) == 32 {
+		copy(headHash[:], val)
+		fmt.Printf("Found migrated head hash at height %d: %x\\n", migratedHeight, headHash)
+		
+		// Set all head pointers
+		rawdb.WriteHeadBlockHash(db, headHash)
+		rawdb.WriteHeadHeaderHash(db, headHash)
+		rawdb.WriteHeadFastBlockHash(db, headHash)
+		rawdb.WriteLastPivotNumber(db, migratedHeight)
+		
+		// Also write the current block pointers
+		rawdb.WriteCanonicalHash(db, headHash, migratedHeight)
+	} else {
+		return nil, fmt.Errorf("could not find canonical hash at height %d", migratedHeight)
+	}
+	
+	// Create blockchain options that skip validation
+	options := &gethcore.BlockChainConfig{
+		TrieCleanLimit: 256,
+		NoPrefetch:     false,
+		StateScheme:    rawdb.HashScheme,
+	}
+	
+	// CRITICAL: Create blockchain WITHOUT genesis
+	// This prevents any genesis initialization
+	fmt.Printf("Creating blockchain without genesis...\\n")
+	blockchain, err := gethcore.NewBlockChain(db, nil, engine, options)
+	if err != nil {
+		fmt.Printf("Failed to create blockchain: %v\\n", err)
+		return nil, fmt.Errorf("failed to create blockchain from migrated data: %w", err)
+	}
+	
+	// Verify the blockchain loaded at the right height
+	currentBlock := blockchain.CurrentBlock()
+	fmt.Printf("Blockchain initialized at height: %d\\n", currentBlock.Number.Uint64())
+	
+	// Create transaction pool
+	legacyPool := legacypool.New(ethconfig.Defaults.TxPool, blockchain)
+	txPool, err := txpool.New(ethconfig.Defaults.TxPool.PriceLimit, blockchain, []txpool.SubPool{legacyPool})
+	if err != nil {
+		return nil, err
+	}
+	
+	return &MinimalEthBackend{
+		chainConfig: chainConfig,
+		blockchain:  blockchain,
+		txPool:      txPool,
+		chainDb:     db,
+		engine:      engine,
+		networkID:   96369,
+	}, nil
+}
+
 // NewMinimalEthBackendForMigration creates a backend that loads from migrated data
 func NewMinimalEthBackendForMigration(db ethdb.Database, config *ethconfig.Config, genesis *gethcore.Genesis, migratedHeight uint64) (*MinimalEthBackend, error) {
 	var chainConfig *params.ChainConfig
`

	if err := os.WriteFile("bypass-genesis.patch", []byte(patch), 0644); err != nil {
		fmt.Printf("Error writing patch: %v\n", err)
		return
	}
	
	fmt.Println("Created bypass-genesis.patch")
	fmt.Println("\nThis patch:")
	fmt.Println("1. Detects migrated data early in VM initialization")
	fmt.Println("2. Creates a special NewMigratedBackend function")
	fmt.Println("3. Completely bypasses genesis initialization")
	fmt.Println("4. Loads blockchain directly from migrated height")
}