package main

import (
	"fmt"
	"os"
)

func main() {
	patch := `--- a/node/vms/cchainvm/vm.go
+++ b/node/vms/cchainvm/vm.go
@@ -85,6 +85,25 @@ func (vm *VM) Initialize(
 	vm.shutdownChan = make(chan struct{})
 	vm.builtBlocks = make(map[ids.ID]*Block)
 	
+	// Create a database wrapper first to check for migrated data
+	vm.ethDB = WrapDatabase(db)
+	
+	// Check for migrated blockchain data BEFORE initializing genesis
+	hasMigratedData := false
+	migratedHeight := uint64(0)
+	if heightBytes, err := vm.ethDB.Get([]byte("Height")); err == nil && len(heightBytes) == 8 {
+		height := binary.BigEndian.Uint64(heightBytes)
+		if height > 0 {
+			hasMigratedData = true
+			migratedHeight = height
+			vm.ctx.Log.Info("Detected migrated blockchain data",
+				zap.Uint64("height", height),
+			)
+		}
+	}
+	
+	// If we have migrated data, skip normal genesis initialization
+	
 	// DEBUG: Log database path and check contents
 	fmt.Printf("DEBUG: C-Chain VM Initialize called\n")
 	fmt.Printf("DEBUG: Database type: %T\n", db)
@@ -125,9 +144,6 @@ func (vm *VM) Initialize(
 		genesis.Config = vm.chainConfig
 	}
 
-	// Create a database wrapper
-	// TODO: Consider using prefixed database in the future
-	vm.ethDB = WrapDatabase(db)
 
 	// Initialize eth config
 	vm.ethConfig = ethconfig.Defaults
@@ -137,7 +153,11 @@ func (vm *VM) Initialize(
 
 	// Create minimal Ethereum backend
 	var err error
-	vm.backend, err = NewMinimalEthBackend(vm.ethDB, &vm.ethConfig, genesis)
+	if hasMigratedData {
+		vm.backend, err = NewMinimalEthBackendForMigration(vm.ethDB, &vm.ethConfig, genesis, migratedHeight)
+	} else {
+		vm.backend, err = NewMinimalEthBackend(vm.ethDB, &vm.ethConfig, genesis)
+	}
 	if err != nil {
 		return fmt.Errorf("failed to create eth backend: %w", err)
 	}
--- a/node/vms/cchainvm/backend.go
+++ b/node/vms/cchainvm/backend.go
@@ -36,6 +36,70 @@ type MinimalEthBackend struct {
 	networkID   uint64
 }
 
+// NewMinimalEthBackendForMigration creates a backend that loads from migrated data
+func NewMinimalEthBackendForMigration(db ethdb.Database, config *ethconfig.Config, genesis *gethcore.Genesis, migratedHeight uint64) (*MinimalEthBackend, error) {
+	chainConfig := genesis.Config
+	if chainConfig == nil {
+		chainConfig = params.AllEthashProtocolChanges
+	}
+
+	// Create consensus engine
+	var engine consensus.Engine
+	if chainConfig.Clique != nil {
+		engine = clique.New(chainConfig.Clique, db)
+	} else {
+		// Use a dummy engine for PoS
+		engine = &dummyEngine{}
+	}
+
+	// Set the head pointers to the migrated height
+	fmt.Printf("Setting blockchain to migrated height %d\n", migratedHeight)
+	
+	// Get the hash at the migrated height
+	blockNumBytes := make([]byte, 8)
+	binary.BigEndian.PutUint64(blockNumBytes, migratedHeight)
+	canonicalKey := append([]byte{0x68}, blockNumBytes...)
+	canonicalKey = append(canonicalKey, 0x6e)
+	
+	var headHash common.Hash
+	if val, err := db.Get(canonicalKey); err == nil && len(val) == 32 {
+		copy(headHash[:], val)
+		fmt.Printf("Found head hash at height %d: %x\n", migratedHeight, headHash)
+		
+		// Write head pointers
+		rawdb.WriteHeadBlockHash(db, headHash)
+		rawdb.WriteHeadHeaderHash(db, headHash)
+		rawdb.WriteHeadFastBlockHash(db, headHash)
+		rawdb.WriteLastPivotNumber(db, migratedHeight)
+	}
+
+	// Initialize blockchain - skip genesis since we have migrated data
+	options := &gethcore.BlockChainConfig{
+		TrieCleanLimit: config.TrieCleanCache,
+		NoPrefetch:     config.NoPrefetch,
+		StateScheme:    rawdb.HashScheme,
+	}
+
+	blockchain, err := gethcore.NewBlockChain(db, nil, engine, options)
+	if err != nil {
+		return nil, fmt.Errorf("failed to create blockchain from migrated data: %w", err)
+	}
+
+	// Create transaction pool
+	legacyPool := legacypool.New(config.TxPool, blockchain)
+	txPool, err := txpool.New(config.TxPool.PriceLimit, blockchain, []txpool.SubPool{legacyPool})
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
+		networkID:   config.NetworkId,
+	}, nil
+}
+
 // NewMinimalEthBackend creates a new minimal Ethereum backend
 func NewMinimalEthBackend(db ethdb.Database, config *ethconfig.Config, genesis *gethcore.Genesis) (*MinimalEthBackend, error) {`

	if err := os.WriteFile("improved-load-migrated.patch", []byte(patch), 0644); err != nil {
		fmt.Printf("Error writing patch: %v\n", err)
		return
	}

	fmt.Println("Created improved-load-migrated.patch")
}
