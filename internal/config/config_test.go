package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	// Save original environment and restore after tests
	originalRaindrop := os.Getenv("RAINDROP_API_TOKEN")
	originalKarakeep := os.Getenv("KARAKEEP_API_TOKEN")
	
	defer func() {
		if originalRaindrop == "" {
			os.Unsetenv("RAINDROP_API_TOKEN")
		} else {
			os.Setenv("RAINDROP_API_TOKEN", originalRaindrop)
		}
		if originalKarakeep == "" {
			os.Unsetenv("KARAKEEP_API_TOKEN")
		} else {
			os.Setenv("KARAKEEP_API_TOKEN", originalKarakeep)
		}
	}()

	tests := []struct {
		name                string
		raindropToken       string
		karakeepToken       string
		expectedRaindrop    string
		expectedKarakeep    string
		setupEnvVars        func()
		cleanupEnvVars      func()
	}{
		{
			name:             "valid tokens from environment",
			raindropToken:    "test-raindrop-token",
			karakeepToken:    "test-karakeep-token",
			expectedRaindrop: "test-raindrop-token",
			expectedKarakeep: "test-karakeep-token",
			setupEnvVars: func() {
				os.Setenv("RAINDROP_API_TOKEN", "test-raindrop-token")
				os.Setenv("KARAKEEP_API_TOKEN", "test-karakeep-token")
			},
			cleanupEnvVars: func() {
				os.Unsetenv("RAINDROP_API_TOKEN")
				os.Unsetenv("KARAKEEP_API_TOKEN")
			},
		},
		{
			name:             "missing tokens",
			expectedRaindrop: "",
			expectedKarakeep: "",
			setupEnvVars: func() {
				os.Unsetenv("RAINDROP_API_TOKEN")
				os.Unsetenv("KARAKEEP_API_TOKEN")
			},
			cleanupEnvVars: func() {},
		},
		{
			name:             "only raindrop token set",
			raindropToken:    "only-raindrop",
			expectedRaindrop: "only-raindrop",
			expectedKarakeep: "",
			setupEnvVars: func() {
				os.Setenv("RAINDROP_API_TOKEN", "only-raindrop")
				os.Unsetenv("KARAKEEP_API_TOKEN")
			},
			cleanupEnvVars: func() {
				os.Unsetenv("RAINDROP_API_TOKEN")
			},
		},
		{
			name:             "only karakeep token set",
			karakeepToken:    "only-karakeep",
			expectedRaindrop: "",
			expectedKarakeep: "only-karakeep",
			setupEnvVars: func() {
				os.Unsetenv("RAINDROP_API_TOKEN")
				os.Setenv("KARAKEEP_API_TOKEN", "only-karakeep")
			},
			cleanupEnvVars: func() {
				os.Unsetenv("KARAKEEP_API_TOKEN")
			},
		},
		{
			name:             "empty string tokens",
			raindropToken:    "",
			karakeepToken:    "",
			expectedRaindrop: "",
			expectedKarakeep: "",
			setupEnvVars: func() {
				os.Setenv("RAINDROP_API_TOKEN", "")
				os.Setenv("KARAKEEP_API_TOKEN", "")
			},
			cleanupEnvVars: func() {
				os.Unsetenv("RAINDROP_API_TOKEN")
				os.Unsetenv("KARAKEEP_API_TOKEN")
			},
		},
		{
			name:             "whitespace only tokens",
			raindropToken:    "   ",
			karakeepToken:    "\t\n ",
			expectedRaindrop: "   ",
			expectedKarakeep: "\t\n ",
			setupEnvVars: func() {
				os.Setenv("RAINDROP_API_TOKEN", "   ")
				os.Setenv("KARAKEEP_API_TOKEN", "\t\n ")
			},
			cleanupEnvVars: func() {
				os.Unsetenv("RAINDROP_API_TOKEN")
				os.Unsetenv("KARAKEEP_API_TOKEN")
			},
		},
		{
			name:             "tokens with special characters",
			raindropToken:    "token-with-!@#$%^&*()_+-=[]{}|;':\",./<>?",
			karakeepToken:    "token.with.dots.and-dashes_and_underscores",
			expectedRaindrop: "token-with-!@#$%^&*()_+-=[]{}|;':\",./<>?",
			expectedKarakeep: "token.with.dots.and-dashes_and_underscores",
			setupEnvVars: func() {
				os.Setenv("RAINDROP_API_TOKEN", "token-with-!@#$%^&*()_+-=[]{}|;':\",./<>?")
				os.Setenv("KARAKEEP_API_TOKEN", "token.with.dots.and-dashes_and_underscores")
			},
			cleanupEnvVars: func() {
				os.Unsetenv("RAINDROP_API_TOKEN")
				os.Unsetenv("KARAKEEP_API_TOKEN")
			},
		},
		{
			name:             "very long token values",
			raindropToken:    strings.Repeat("a", 1000),
			karakeepToken:    strings.Repeat("b", 2000),
			expectedRaindrop: strings.Repeat("a", 1000),
			expectedKarakeep: strings.Repeat("b", 2000),
			setupEnvVars: func() {
				os.Setenv("RAINDROP_API_TOKEN", strings.Repeat("a", 1000))
				os.Setenv("KARAKEEP_API_TOKEN", strings.Repeat("b", 2000))
			},
			cleanupEnvVars: func() {
				os.Unsetenv("RAINDROP_API_TOKEN")
				os.Unsetenv("KARAKEEP_API_TOKEN")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			tt.setupEnvVars()
			defer tt.cleanupEnvVars()

			// Test
			cfg, err := Load()

			// Assertions
			if err != nil {
				t.Errorf("Load() error = %v, expected nil", err)
				return
			}

			if cfg.RaindropToken != tt.expectedRaindrop {
				t.Errorf("Load() RaindropToken = %q, expected %q", cfg.RaindropToken, tt.expectedRaindrop)
			}

			if cfg.KarakeepToken != tt.expectedKarakeep {
				t.Errorf("Load() KarakeepToken = %q, expected %q", cfg.KarakeepToken, tt.expectedKarakeep)
			}
		})
	}
}

func TestLoadFromEnvFile(t *testing.T) {
	// Save original environment and restore after test
	originalRaindrop := os.Getenv("RAINDROP_API_TOKEN")
	originalKarakeep := os.Getenv("KARAKEEP_API_TOKEN")
	
	defer func() {
		if originalRaindrop == "" {
			os.Unsetenv("RAINDROP_API_TOKEN")
		} else {
			os.Setenv("RAINDROP_API_TOKEN", originalRaindrop)
		}
		if originalKarakeep == "" {
			os.Unsetenv("KARAKEEP_API_TOKEN")
		} else {
			os.Setenv("KARAKEEP_API_TOKEN", originalKarakeep)
		}
	}()

	tests := []struct {
		name             string
		envFileContent   string
		expectedRaindrop string
		expectedKarakeep string
	}{
		{
			name: "valid .env file",
			envFileContent: `RAINDROP_API_TOKEN=env-file-raindrop-token
KARAKEEP_API_TOKEN=env-file-karakeep-token`,
			expectedRaindrop: "env-file-raindrop-token",
			expectedKarakeep: "env-file-karakeep-token",
		},
		{
			name: "partial .env file - only raindrop",
			envFileContent: `RAINDROP_API_TOKEN=only-raindrop-in-file`,
			expectedRaindrop: "only-raindrop-in-file",
			expectedKarakeep: "",
		},
		{
			name: "partial .env file - only karakeep",
			envFileContent: `KARAKEEP_API_TOKEN=only-karakeep-in-file`,
			expectedRaindrop: "",
			expectedKarakeep: "only-karakeep-in-file",
		},
		{
			name: ".env file with empty values",
			envFileContent: `RAINDROP_API_TOKEN=
KARAKEEP_API_TOKEN=`,
			expectedRaindrop: "",
			expectedKarakeep: "",
		},
		{
			name: ".env file with quoted values",
			envFileContent: `RAINDROP_API_TOKEN="quoted-raindrop-token"
KARAKEEP_API_TOKEN='single-quoted-karakeep'`,
			expectedRaindrop: "quoted-raindrop-token",
			expectedKarakeep: "single-quoted-karakeep",
		},
		{
			name: ".env file with special characters and spaces",
			envFileContent: `RAINDROP_API_TOKEN=token with spaces and special chars !@#$%
KARAKEEP_API_TOKEN="token_with_underscores_and-dashes.dots"`,
			expectedRaindrop: "token with spaces and special chars !@#$%",
			expectedKarakeep: "token_with_underscores_and-dashes.dots",
		},
		{
			name: ".env file with comments and extra whitespace",
			envFileContent: `# This is a comment
RAINDROP_API_TOKEN=  raindrop-with-spaces  
  KARAKEEP_API_TOKEN=karakeep-token  # inline comment`,
			expectedRaindrop: "raindrop-with-spaces",
			expectedKarakeep: "karakeep-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for test
			tempDir := t.TempDir()
			envFile := filepath.Join(tempDir, ".env")

			// Write .env file
			if err := os.WriteFile(envFile, []byte(tt.envFileContent), 0644); err != nil {
				t.Fatalf("Failed to create .env file: %v", err)
			}

			// Change to temporary directory so .env file is found
			originalDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get current directory: %v", err)
			}
			defer os.Chdir(originalDir)

			if err := os.Chdir(tempDir); err != nil {
				t.Fatalf("Failed to change to temp directory: %v", err)
			}

			// Clear environment variables to ensure we're only testing .env file
			os.Unsetenv("RAINDROP_API_TOKEN")
			os.Unsetenv("KARAKEEP_API_TOKEN")

			// Test
			cfg, err := Load()

			// Assertions
			if err != nil {
				t.Errorf("Load() error = %v, expected nil", err)
				return
			}

			if cfg.RaindropToken != tt.expectedRaindrop {
				t.Errorf("Load() RaindropToken = %q, expected %q", cfg.RaindropToken, tt.expectedRaindrop)
			}

			if cfg.KarakeepToken != tt.expectedKarakeep {
				t.Errorf("Load() KarakeepToken = %q, expected %q", cfg.KarakeepToken, tt.expectedKarakeep)
			}
		})
	}
}

func TestEnvironmentVariablePrecedenceOverEnvFile(t *testing.T) {
	// Save original environment and restore after test
	originalRaindrop := os.Getenv("RAINDROP_API_TOKEN")
	originalKarakeep := os.Getenv("KARAKEEP_API_TOKEN")
	
	defer func() {
		if originalRaindrop == "" {
			os.Unsetenv("RAINDROP_API_TOKEN")
		} else {
			os.Setenv("RAINDROP_API_TOKEN", originalRaindrop)
		}
		if originalKarakeep == "" {
			os.Unsetenv("KARAKEEP_API_TOKEN")
		} else {
			os.Setenv("KARAKEEP_API_TOKEN", originalKarakeep)
		}
	}()

	// Create temporary directory for test
	tempDir := t.TempDir()
	envFile := filepath.Join(tempDir, ".env")

	// Write .env file with specific values
	envContent := `RAINDROP_API_TOKEN=env-file-raindrop
KARAKEEP_API_TOKEN=env-file-karakeep`
	
	if err := os.WriteFile(envFile, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	// Change to temporary directory so .env file is found
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	tests := []struct {
		name                    string
		envRaindropToken        string
		envKarakeepToken        string
		setRaindropEnv          bool
		setKarakeepEnv          bool
		expectedRaindropToken   string
		expectedKarakeepToken   string
	}{
		{
			name:                  "environment variables override .env file",
			envRaindropToken:      "env-var-raindrop",
			envKarakeepToken:      "env-var-karakeep",
			setRaindropEnv:        true,
			setKarakeepEnv:        true,
			expectedRaindropToken: "env-var-raindrop",
			expectedKarakeepToken: "env-var-karakeep",
		},
		{
			name:                  "only raindrop env var set, karakeep from .env",
			envRaindropToken:      "env-var-raindrop-only",
			setRaindropEnv:        true,
			setKarakeepEnv:        false,
			expectedRaindropToken: "env-var-raindrop-only",
			expectedKarakeepToken: "env-file-karakeep",
		},
		{
			name:                  "only karakeep env var set, raindrop from .env",
			envKarakeepToken:      "env-var-karakeep-only",
			setRaindropEnv:        false,
			setKarakeepEnv:        true,
			expectedRaindropToken: "env-file-raindrop",
			expectedKarakeepToken: "env-var-karakeep-only",
		},
		{
			name:                  "empty env vars override .env file",
			envRaindropToken:      "",
			envKarakeepToken:      "",
			setRaindropEnv:        true,
			setKarakeepEnv:        true,
			expectedRaindropToken: "",
			expectedKarakeepToken: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment variables
			if tt.setRaindropEnv {
				os.Setenv("RAINDROP_API_TOKEN", tt.envRaindropToken)
			} else {
				os.Unsetenv("RAINDROP_API_TOKEN")
			}

			if tt.setKarakeepEnv {
				os.Setenv("KARAKEEP_API_TOKEN", tt.envKarakeepToken)
			} else {
				os.Unsetenv("KARAKEEP_API_TOKEN")
			}

			// Cleanup after test
			defer func() {
				os.Unsetenv("RAINDROP_API_TOKEN")
				os.Unsetenv("KARAKEEP_API_TOKEN")
			}()

			// Test
			cfg, err := Load()

			// Assertions
			if err != nil {
				t.Errorf("Load() error = %v, expected nil", err)
				return
			}

			if cfg.RaindropToken != tt.expectedRaindropToken {
				t.Errorf("Load() RaindropToken = %q, expected %q", cfg.RaindropToken, tt.expectedRaindropToken)
			}

			if cfg.KarakeepToken != tt.expectedKarakeepToken {
				t.Errorf("Load() KarakeepToken = %q, expected %q", cfg.KarakeepToken, tt.expectedKarakeepToken)
			}
		})
	}
}

func TestLoadWithMissingEnvFile(t *testing.T) {
	// Save original environment and restore after test
	originalRaindrop := os.Getenv("RAINDROP_API_TOKEN")
	originalKarakeep := os.Getenv("KARAKEEP_API_TOKEN")
	
	defer func() {
		if originalRaindrop == "" {
			os.Unsetenv("RAINDROP_API_TOKEN")
		} else {
			os.Setenv("RAINDROP_API_TOKEN", originalRaindrop)
		}
		if originalKarakeep == "" {
			os.Unsetenv("KARAKEEP_API_TOKEN")
		} else {
			os.Setenv("KARAKEEP_API_TOKEN", originalKarakeep)
		}
	}()

	// Create temporary directory without .env file
	tempDir := t.TempDir()

	// Change to temporary directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Set environment variables
	os.Setenv("RAINDROP_API_TOKEN", "env-raindrop")
	os.Setenv("KARAKEEP_API_TOKEN", "env-karakeep")

	defer func() {
		os.Unsetenv("RAINDROP_API_TOKEN")
		os.Unsetenv("KARAKEEP_API_TOKEN")
	}()

	// Test
	cfg, err := Load()

	// Assertions
	if err != nil {
		t.Errorf("Load() error = %v, expected nil", err)
		return
	}

	if cfg.RaindropToken != "env-raindrop" {
		t.Errorf("Load() RaindropToken = %q, expected %q", cfg.RaindropToken, "env-raindrop")
	}

	if cfg.KarakeepToken != "env-karakeep" {
		t.Errorf("Load() KarakeepToken = %q, expected %q", cfg.KarakeepToken, "env-karakeep")
	}
}

// TestConfigStruct tests the Config struct itself
func TestConfigStruct(t *testing.T) {
	cfg := &Config{
		RaindropToken: "test-raindrop",
		KarakeepToken: "test-karakeep",
	}

	if cfg.RaindropToken != "test-raindrop" {
		t.Errorf("Config.RaindropToken = %q, expected %q", cfg.RaindropToken, "test-raindrop")
	}

	if cfg.KarakeepToken != "test-karakeep" {
		t.Errorf("Config.KarakeepToken = %q, expected %q", cfg.KarakeepToken, "test-karakeep")
	}
}

// Benchmark tests for performance characteristics
func BenchmarkLoad(b *testing.B) {
	// Setup
	os.Setenv("RAINDROP_API_TOKEN", "benchmark-raindrop-token")
	os.Setenv("KARAKEEP_API_TOKEN", "benchmark-karakeep-token")
	
	defer func() {
		os.Unsetenv("RAINDROP_API_TOKEN")
		os.Unsetenv("KARAKEEP_API_TOKEN")
	}()

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		cfg, err := Load()
		if err != nil {
			b.Fatalf("Load() error = %v", err)
		}
		if cfg.RaindropToken == "" || cfg.KarakeepToken == "" {
			b.Fatal("Expected tokens to be loaded")
		}
	}
}

func BenchmarkLoadWithLargeTokens(b *testing.B) {
	// Setup with very large tokens
	largeToken := strings.Repeat("abcdefghijklmnopqrstuvwxyz", 1000) // 26,000 characters
	os.Setenv("RAINDROP_API_TOKEN", largeToken)
	os.Setenv("KARAKEEP_API_TOKEN", largeToken)
	
	defer func() {
		os.Unsetenv("RAINDROP_API_TOKEN")
		os.Unsetenv("KARAKEEP_API_TOKEN")
	}()

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		cfg, err := Load()
		if err != nil {
			b.Fatalf("Load() error = %v", err)
		}
		if len(cfg.RaindropToken) != 26000 || len(cfg.KarakeepToken) != 26000 {
			b.Fatal("Expected large tokens to be loaded")
		}
	}
}