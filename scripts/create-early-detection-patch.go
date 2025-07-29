package main

import (
	"fmt"
	"os"
)

func main() {
	patch := `--- a/node/vms/cchainvm/vm.go
+++ b/node/vms/cchainvm/vm.go
@@ -85,11 +85,21 @@ func (vm *VM) Initialize(
 	vm.shutdownChan = make(chan struct{})
 	vm.builtBlocks = make(map[ids.ID]*Block)
 	
-	// Create a database wrapper first to check for migrated data
-	vm.ethDB = WrapDatabase(db)
+	// Check for migrated data BEFORE wrapping the database
+	hasMigratedData := false
+	migratedHeight := uint64(0)
+	
+	// Try to read Height key directly from raw db
+	if rawDB, ok := db.(database.Database); ok {
+		if heightBytes, err := rawDB.Get([]byte("Height")); err == nil && len(heightBytes) == 8 {
+			migratedHeight = binary.BigEndian.Uint64(heightBytes)
+			hasMigratedData = migratedHeight > 0
+			fmt.Printf("DEBUG: Found migrated data at height %d\n", migratedHeight)
+		}
+	}
 	
-	// Check for migrated blockchain data BEFORE initializing genesis
-	hasMigratedData := false
-	migratedHeight := uint64(0)
-	if heightBytes, err := vm.ethDB.Get([]byte("Height")); err == nil && len(heightBytes) == 8 {
-		height := binary.BigEndian.Uint64(heightBytes)
-		if height > 0 {
-			hasMigratedData = true
-			migratedHeight = height
-			vm.ctx.Log.Info("Detected migrated blockchain data",
-				zap.Uint64("height", height),
-			)
-		}
-	}
+	// Now create the wrapped database
+	vm.ethDB = WrapDatabase(db)
 	
 	// If we have migrated data, skip normal genesis initialization
 	
`

	if err := os.WriteFile("early-detection.patch", []byte(patch), 0644); err != nil {
		fmt.Printf("Error writing patch: %v\n", err)
		return
	}
	
	fmt.Println("Created early-detection.patch")
}