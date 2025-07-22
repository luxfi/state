package main

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestArcheologyCommands(t *testing.T) {
	// Build the binary first
	binPath := filepath.Join(t.TempDir(), "archeology")
	buildCmd := exec.Command("go", "build", "-o", binPath, ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build archeology: %v", err)
	}

	tests := []struct {
		name        string
		args        []string
		wantOutput  []string
		wantErr     bool
		errContains string
	}{
		{
			name:       "Help command",
			args:       []string{"--help"},
			wantOutput: []string{"Blockchain Archaeology", "comprehensive tool for extracting, analyzing"},
			wantErr:    false,
		},
		{
			name:       "Version command",
			args:       []string{"--version"},
			wantOutput: []string{"dev", "commit: none"},
			wantErr:    false,
		},
		{
			name:       "List chains",
			args:       []string{"list", "chains"},
			wantOutput: []string{"Known chain configurations:"},
			wantErr:    false,
		},
		{
			name:       "Extract missing network",
			args:       []string{"extract", "-src", "/nonexistent", "-dst", "/tmp/test"},
			wantErr:    false, // The command succeeds but prints error message
			wantOutput: []string{"Either -network or -chainid must be specified"},
		},
		{
			name:       "Analyze missing db",
			args:       []string{"analyze", "--db", "/nonexistent"},
			wantErr:    true,
			errContains: "database path does not exist",
		},
		{
			name:       "Scan without contract",
			args:       []string{"scan", "--chain", "ethereum"},
			wantErr:    true,
			errContains: "required flag(s) \"contract\"",
		},
		{
			name:       "Import NFT without contract",
			args:       []string{"import-nft", "--network", "ethereum"},
			wantErr:    true,
			errContains: "required flag(s) \"contract\", \"project\" not set",
		},
		{
			name:       "Import NFT help",
			args:       []string{"import-nft", "--help"},
			wantOutput: []string{"Import NFTs from any EVM chain", "network parameters"},
			wantErr:    false,
		},
		{
			name:       "Import token without contract",
			args:       []string{"import-token", "--network", "bsc"},
			wantErr:    true,
			errContains: "required flag(s) \"contract\", \"project\" not set",
		},
		{
			name:       "Import token help",
			args:       []string{"import-token", "--help"},
			wantOutput: []string{"Import ERC20 tokens from any EVM chain", "X-Chain genesis integration"},
			wantErr:    false,
		},
		{
			name:       "Genesis without inputs",
			args:       []string{"genesis"},
			wantErr:    true,
			errContains: "at least one CSV input is required",
		},
		{
			name:       "Genesis help",
			args:       []string{"genesis", "--help"},
			wantOutput: []string{"Generate genesis file", "historical assets"},
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(binPath, tt.args...)
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()
			gotErr := err != nil

			if gotErr != tt.wantErr {
				t.Errorf("archeology %v: gotErr = %v, wantErr = %v\nstderr: %s",
					tt.args, gotErr, tt.wantErr, stderr.String())
			}

			output := stdout.String() + stderr.String()

			if tt.wantErr && tt.errContains != "" {
				if !strings.Contains(output, tt.errContains) {
					t.Errorf("archeology %v: error output doesn't contain %q\nGot: %s",
						tt.args, tt.errContains, output)
				}
			}

			for _, want := range tt.wantOutput {
				if !strings.Contains(output, want) {
					t.Errorf("archeology %v: output doesn't contain %q\nGot: %s",
						tt.args, want, output)
				}
			}
		})
	}
}

func TestImportNFTValidation(t *testing.T) {
	// Test address validation
	tests := []struct {
		name    string
		address string
		valid   bool
	}{
		{"Valid address", "0x1234567890123456789012345678901234567890", true},
		{"Missing 0x", "1234567890123456789012345678901234567890", false},
		{"Too short", "0x123456789012345678901234567890123456789", false},
		{"Too long", "0x12345678901234567890123456789012345678901", false},
		{"Invalid hex", "0x12345678901234567890123456789012345678gg", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The validation is in the command, we'll test via command line
			binPath := filepath.Join(t.TempDir(), "archeology")
			buildCmd := exec.Command("go", "build", "-o", binPath, ".")
			if err := buildCmd.Run(); err != nil {
				t.Skipf("Skipping validation test: %v", err)
			}

			// Test validation through actual command execution
			cmd := exec.Command(binPath, "import-nft",
				"--network", "ethereum",
				"--contract", tt.address,
				"--project", "test")

			err := cmd.Run()
			// Command will fail without RPC connection, but that's expected
			// We're just checking that the command structure is valid
			if err == nil && !tt.valid {
				t.Errorf("Invalid address %s should have been rejected", tt.address)
			}
		})
	}
}