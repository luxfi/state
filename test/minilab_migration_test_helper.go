package main

import (
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/gomega"
)

// EnsureMigratedDB ensures the migrated database exists
func EnsureMigratedDB(projectRoot, srcDB, migratedDB string) {
	if _, err := os.Stat(migratedDB); os.IsNotExist(err) {
		cmd := exec.Command(
			filepath.Join(projectRoot, "bin", "migrate_subset"),
			"--src", srcDB,
			"--dst", migratedDB,
			"--limit", "50000",
		)
		output, err := cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), string(output))
	}
}
