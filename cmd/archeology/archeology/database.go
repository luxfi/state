package archeology

import (
	"fmt"
	"os"
	"path/filepath"
)

// DatabaseType represents the type of database
type DatabaseType int

const (
	DBTypeUnknown DatabaseType = iota
	DBTypeLevelDB
	DBTypePebbleDB
)

// String returns the string representation of the database type
func (t DatabaseType) String() string {
	switch t {
	case DBTypeLevelDB:
		return "LevelDB"
	case DBTypePebbleDB:
		return "PebbleDB"
	default:
		return "Unknown"
	}
}

// DetectDatabaseType detects the type of database at the given path
func DetectDatabaseType(path string) (DatabaseType, error) {
	// Check if path exists
	if _, err := os.Stat(path); err != nil {
		return DBTypeUnknown, fmt.Errorf("database path does not exist: %w", err)
	}

	// Check for PebbleDB markers
	if _, err := os.Stat(filepath.Join(path, "MANIFEST-000001")); err == nil {
		// Check for PebbleDB-specific files
		if _, err := os.Stat(filepath.Join(path, "OPTIONS-000003")); err == nil {
			return DBTypePebbleDB, nil
		}
	}

	// Check for LevelDB markers
	if _, err := os.Stat(filepath.Join(path, "LOG")); err == nil {
		if _, err := os.Stat(filepath.Join(path, "CURRENT")); err == nil {
			return DBTypeLevelDB, nil
		}
	}

	// Check for ancient directory (geth-style)
	if _, err := os.Stat(filepath.Join(path, "ancient")); err == nil {
		return DBTypeLevelDB, nil
	}

	return DBTypeUnknown, fmt.Errorf("unable to determine database type")
}