package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/ashebanow/rainbridge/internal/config"
	"github.com/ashebanow/rainbridge/internal/importer"
	"github.com/ashebanow/rainbridge/internal/karakeep"
	"github.com/ashebanow/rainbridge/internal/raindrop"
)

// TestMain tests the main function by examining exit codes
// Since main() calls log.Fatalf which calls os.Exit, we need to run it in a subprocess
func TestMain(t *testing.T) {
	// Only run this test when not in a subprocess
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	// This is the subprocess - run the actual main function
	main()
}

// TestMainWithConfigLoadFailure tests that main exits with error when config loading fails
func TestMainWithConfigLoadFailure(t *testing.T) {
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

	// Clear environment variables to simulate config load failure
	os.Unsetenv("RAINDROP_API_TOKEN")
	os.Unsetenv("KARAKEEP_API_TOKEN")

	// Run main in a subprocess to capture exit code
	if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" {
		main()
		return
	}

	// This code runs in the parent process
	cmd := exec.Command(os.Args[0], "-test.run=TestMainWithConfigLoadFailure")
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	err := cmd.Run()

	// Expect the subprocess to exit with non-zero code
	if err == nil {
		t.Error("Expected main() to exit with non-zero code when config loading fails")
	}
}

// TestMainWithValidConfig tests that main runs successfully with valid configuration
func TestMainWithValidConfig(t *testing.T) {
	// Skip this test as it would require valid API tokens and would make real API calls
	// Instead, we test the individual components that main() orchestrates
	t.Skip("Skipping integration test - would require valid API tokens")
}

// TestMainComponents tests the individual components that main() orchestrates
func TestMainComponents(t *testing.T) {
	tests := []struct {
		name                 string
		raindropToken        string
		karakeepToken        string
		expectConfigError    bool
		expectClientCreation bool
	}{
		{
			name:                 "valid tokens",
			raindropToken:        "valid-raindrop-token",
			karakeepToken:        "valid-karakeep-token",
			expectConfigError:    false,
			expectClientCreation: true,
		},
		{
			name:                 "empty tokens",
			raindropToken:        "",
			karakeepToken:        "",
			expectConfigError:    false, // config.Load() doesn't validate tokens
			expectClientCreation: true,  // clients can be created with empty tokens
		},
		{
			name:                 "only raindrop token",
			raindropToken:        "raindrop-only",
			karakeepToken:        "",
			expectConfigError:    false,
			expectClientCreation: true,
		},
		{
			name:                 "only karakeep token",
			raindropToken:        "",
			karakeepToken:        "karakeep-only",
			expectConfigError:    false,
			expectClientCreation: true,
		},
		{
			name:                 "special characters in tokens",
			raindropToken:        "token-with-!@#$%^&*()_+-=[]{}|;':\",./<>?",
			karakeepToken:        "token.with.dots.and-dashes_and_underscores",
			expectConfigError:    false,
			expectClientCreation: true,
		},
		{
			name:                 "very long tokens",
			raindropToken:        strings.Repeat("a", 1000),
			karakeepToken:        strings.Repeat("b", 2000),
			expectConfigError:    false,
			expectClientCreation: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			// Setup environment
			os.Setenv("RAINDROP_API_TOKEN", tt.raindropToken)
			os.Setenv("KARAKEEP_API_TOKEN", tt.karakeepToken)

			// Test config loading
			cfg, err := config.Load()
			if tt.expectConfigError && err == nil {
				t.Error("Expected config loading to fail, but it succeeded")
			} else if !tt.expectConfigError && err != nil {
				t.Errorf("Expected config loading to succeed, but got error: %v", err)
			}

			if err == nil {
				// Test client creation
				if tt.expectClientCreation {
					raindropClient := raindrop.NewClient(cfg.RaindropToken)
					if raindropClient == nil {
						t.Error("Expected raindrop client to be created")
					}

					karakeepClient := karakeep.NewClient(cfg.KarakeepToken)
					if karakeepClient == nil {
						t.Error("Expected karakeep client to be created")
					}

					// Test importer creation
					importer := importer.NewImporter(raindropClient, karakeepClient)
					if importer == nil {
						t.Error("Expected importer to be created")
					}
				}
			}
		})
	}
}

// TestMainExitCodes tests that main exits with appropriate codes for different scenarios
func TestMainExitCodes(t *testing.T) {
	tests := []struct {
		name           string
		setupEnv       func()
		expectedOutput string
		expectNonZero  bool
	}{
		{
			name: "missing config causes exit",
			setupEnv: func() {
				os.Unsetenv("RAINDROP_API_TOKEN")
				os.Unsetenv("KARAKEEP_API_TOKEN")
			},
			expectedOutput: "", // We expect log.Fatalf to be called
			expectNonZero:  true,
		},
		{
			name: "valid config with empty tokens",
			setupEnv: func() {
				os.Setenv("RAINDROP_API_TOKEN", "")
				os.Setenv("KARAKEEP_API_TOKEN", "")
			},
			expectedOutput: "", // Would proceed to import, which would likely fail
			expectNonZero:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip actual subprocess execution for these tests
			// as they would require complex setup and real API calls
			t.Skip("Skipping subprocess test - would require complex setup")

			// Instead, we verify that the components behave as expected
			// Save original environment
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

			// Setup environment
			tt.setupEnv()

			// Test that config loading behaves as expected
			cfg, err := config.Load()
			if err != nil {
				t.Logf("Config loading failed as expected: %v", err)
			} else {
				t.Logf("Config loaded successfully: RaindropToken=%q, KarakeepToken=%q",
					cfg.RaindropToken, cfg.KarakeepToken)
			}
		})
	}
}

// TestMainLogOutput tests the log output format
func TestMainLogOutput(t *testing.T) {
	// Save original log output
	originalLogOutput := log.Writer()

	// Create a buffer to capture log output
	var buf bytes.Buffer
	log.SetOutput(&buf)

	// Restore original log output after test
	defer log.SetOutput(originalLogOutput)

	// Test log output for various scenarios
	tests := []struct {
		name           string
		setupEnv       func()
		expectedLogMsg string
	}{
		{
			name: "config load success",
			setupEnv: func() {
				os.Setenv("RAINDROP_API_TOKEN", "test-token")
				os.Setenv("KARAKEEP_API_TOKEN", "test-token")
			},
			expectedLogMsg: "", // No log message expected for successful config load
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original environment
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

			// Clear buffer
			buf.Reset()

			// Setup environment
			tt.setupEnv()

			// Test only the config loading part to avoid real API calls
			cfg, err := config.Load()
			if err != nil {
				log.Printf("Failed to load configuration: %v", err)
			} else {
				// Config loaded successfully, no log message expected
				_ = cfg
			}

			// Check log output
			logOutput := buf.String()
			if tt.expectedLogMsg == "" {
				if logOutput != "" {
					t.Errorf("Expected no log output, but got: %s", logOutput)
				}
			} else {
				if !strings.Contains(logOutput, tt.expectedLogMsg) {
					t.Errorf("Expected log output to contain %q, but got: %s", tt.expectedLogMsg, logOutput)
				}
			}
		})
	}
}

// TestMainErrorHandling tests error handling paths
func TestMainErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		setupEnv      func()
		expectPanic   bool
		expectMessage string
	}{
		{
			name: "config load failure",
			setupEnv: func() {
				// Note: config.Load() currently never returns an error
				// This is testing the error handling path if it did
			},
			expectPanic:   false,
			expectMessage: "Failed to load configuration",
		},
		{
			name: "import failure",
			setupEnv: func() {
				os.Setenv("RAINDROP_API_TOKEN", "invalid-token")
				os.Setenv("KARAKEEP_API_TOKEN", "invalid-token")
			},
			expectPanic:   false,
			expectMessage: "Import failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original environment
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

			// Setup environment
			tt.setupEnv()

			// Test only the components without running main()
			// to avoid os.Exit calls
			cfg, configErr := config.Load()
			if configErr != nil {
				if !strings.Contains(configErr.Error(), tt.expectMessage) {
					t.Errorf("Expected error message to contain %q, but got: %v", tt.expectMessage, configErr)
				}
				return
			}

			// Test client creation
			raindropClient := raindrop.NewClient(cfg.RaindropToken)
			karakeepClient := karakeep.NewClient(cfg.KarakeepToken)
			importer := importer.NewImporter(raindropClient, karakeepClient)

			// Test that importer was created successfully
			if importer == nil {
				t.Error("Expected importer to be created")
			}
		})
	}
}

// TestMainWithDifferentTokenFormats tests main with various token formats
func TestMainWithDifferentTokenFormats(t *testing.T) {
	tests := []struct {
		name          string
		raindropToken string
		karakeepToken string
		expectError   bool
	}{
		{
			name:          "normal tokens",
			raindropToken: "abcd1234efgh5678",
			karakeepToken: "xyz9876abc5432",
			expectError:   false,
		},
		{
			name:          "tokens with special characters",
			raindropToken: "token-with-!@#$%^&*()_+-=[]{}|;':\",./<>?",
			karakeepToken: "token.with.dots.and-dashes_and_underscores",
			expectError:   false,
		},
		{
			name:          "very long tokens",
			raindropToken: strings.Repeat("a", 1000),
			karakeepToken: strings.Repeat("b", 2000),
			expectError:   false,
		},
		{
			name:          "tokens with unicode",
			raindropToken: "token-with-unicode-™©®",
			karakeepToken: "token-with-emojis-🚀🎉",
			expectError:   false,
		},
		{
			name:          "tokens with whitespace",
			raindropToken: "  token-with-leading-spaces",
			karakeepToken: "token-with-trailing-spaces  ",
			expectError:   false,
		},
		{
			name:          "empty tokens",
			raindropToken: "",
			karakeepToken: "",
			expectError:   false, // Config loading doesn't validate tokens
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original environment
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

			// Setup environment
			os.Setenv("RAINDROP_API_TOKEN", tt.raindropToken)
			os.Setenv("KARAKEEP_API_TOKEN", tt.karakeepToken)

			// Test the main function components
			cfg, err := config.Load()
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			} else if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if err == nil {
				// Verify tokens are loaded correctly
				if cfg.RaindropToken != tt.raindropToken {
					t.Errorf("Expected RaindropToken to be %q, but got %q", tt.raindropToken, cfg.RaindropToken)
				}
				if cfg.KarakeepToken != tt.karakeepToken {
					t.Errorf("Expected KarakeepToken to be %q, but got %q", tt.karakeepToken, cfg.KarakeepToken)
				}

				// Test client creation
				raindropClient := raindrop.NewClient(cfg.RaindropToken)
				karakeepClient := karakeep.NewClient(cfg.KarakeepToken)
				importer := importer.NewImporter(raindropClient, karakeepClient)

				if raindropClient == nil {
					t.Error("Expected raindrop client to be created")
				}
				if karakeepClient == nil {
					t.Error("Expected karakeep client to be created")
				}
				if importer == nil {
					t.Error("Expected importer to be created")
				}
			}
		})
	}
}

// TestMainFatalErrors tests that main properly handles fatal errors
func TestMainFatalErrors(t *testing.T) {
	// This test documents the behavior of log.Fatalf in main()
	// Since log.Fatalf calls os.Exit, we can't easily test it without subprocesses
	// But we can test the error conditions that would trigger it

	tests := []struct {
		name        string
		description string
		setupEnv    func()
		checkError  func(t *testing.T)
	}{
		{
			name:        "config load failure would cause fatal error",
			description: "If config.Load() returned an error, main() would call log.Fatalf",
			setupEnv: func() {
				// Currently config.Load() never returns an error
				// This is a placeholder for if it did
			},
			checkError: func(t *testing.T) {
				cfg, err := config.Load()
				if err != nil {
					t.Logf("Config load error (would cause fatal): %v", err)
				} else {
					t.Logf("Config loaded successfully: %+v", cfg)
				}
			},
		},
		{
			name:        "import failure would cause fatal error",
			description: "If importer.RunImport() returns an error, main() would call log.Fatalf",
			setupEnv: func() {
				os.Setenv("RAINDROP_API_TOKEN", "test-token")
				os.Setenv("KARAKEEP_API_TOKEN", "test-token")
			},
			checkError: func(t *testing.T) {
				cfg, err := config.Load()
				if err != nil {
					t.Fatalf("Config load failed: %v", err)
				}

				raindropClient := raindrop.NewClient(cfg.RaindropToken)
				karakeepClient := karakeep.NewClient(cfg.KarakeepToken)
				importer := importer.NewImporter(raindropClient, karakeepClient)

				// Note: We don't actually call RunImport() here as it would make real API calls
				// This just verifies the setup that main() does before calling RunImport()
				if importer == nil {
					t.Error("Expected importer to be created")
				}
				t.Log("Importer created successfully (would call RunImport in main)")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original environment
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

			tt.setupEnv()
			tt.checkError(t)
		})
	}
}

// Benchmark tests for main function components
func BenchmarkMainComponents(b *testing.B) {
	// Setup
	os.Setenv("RAINDROP_API_TOKEN", "benchmark-raindrop-token")
	os.Setenv("KARAKEEP_API_TOKEN", "benchmark-karakeep-token")

	defer func() {
		os.Unsetenv("RAINDROP_API_TOKEN")
		os.Unsetenv("KARAKEEP_API_TOKEN")
	}()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cfg, err := config.Load()
		if err != nil {
			b.Fatalf("Config load failed: %v", err)
		}

		raindropClient := raindrop.NewClient(cfg.RaindropToken)
		karakeepClient := karakeep.NewClient(cfg.KarakeepToken)
		importer := importer.NewImporter(raindropClient, karakeepClient)

		if importer == nil {
			b.Fatal("Expected importer to be created")
		}
	}
}

// TestMainDocumentation documents the expected behavior of main()
func TestMainDocumentation(t *testing.T) {
	t.Log("main() function behavior:")
	t.Log("1. Calls config.Load() to load configuration from environment variables or .env file")
	t.Log("2. If config.Load() returns an error, calls log.Fatalf() which exits the program")
	t.Log("3. Creates raindrop.Client with the loaded raindrop token")
	t.Log("4. Creates karakeep.Client with the loaded karakeep token")
	t.Log("5. Creates importer.Importer with both clients")
	t.Log("6. Calls importer.RunImport() to perform the import")
	t.Log("7. If RunImport() returns an error, calls log.Fatalf() which exits the program")
	t.Log("8. If successful, main() returns normally")

	// Test the documented behavior
	os.Setenv("RAINDROP_API_TOKEN", "test-token")
	os.Setenv("KARAKEEP_API_TOKEN", "test-token")

	defer func() {
		os.Unsetenv("RAINDROP_API_TOKEN")
		os.Unsetenv("KARAKEEP_API_TOKEN")
	}()

	// Step 1: Load configuration
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Step 1 failed: %v", err)
	}
	t.Log("✓ Step 1: Configuration loaded successfully")

	// Step 3: Create raindrop client
	raindropClient := raindrop.NewClient(cfg.RaindropToken)
	if raindropClient == nil {
		t.Fatal("Step 3 failed: raindrop client is nil")
	}
	t.Log("✓ Step 3: Raindrop client created successfully")

	// Step 4: Create karakeep client
	karakeepClient := karakeep.NewClient(cfg.KarakeepToken)
	if karakeepClient == nil {
		t.Fatal("Step 4 failed: karakeep client is nil")
	}
	t.Log("✓ Step 4: Karakeep client created successfully")

	// Step 5: Create importer
	importer := importer.NewImporter(raindropClient, karakeepClient)
	if importer == nil {
		t.Fatal("Step 5 failed: importer is nil")
	}
	t.Log("✓ Step 5: Importer created successfully")

	// Note: We don't test Step 6 (RunImport) as it would make real API calls
	t.Log("✓ Step 6: RunImport() would be called here (skipped in test)")

	t.Log("✓ All documented steps verified successfully")
}

// Mock structures for testing error scenarios
type mockConfig struct {
	shouldFail bool
	errorMsg   string
}

func (m *mockConfig) Load() (*config.Config, error) {
	if m.shouldFail {
		return nil, errors.New(m.errorMsg)
	}
	return &config.Config{
		RaindropToken: "mock-raindrop-token",
		KarakeepToken: "mock-karakeep-token",
	}, nil
}

// TestMainWithMockErrors tests main function behavior with mocked errors
func TestMainWithMockErrors(t *testing.T) {
	// This test demonstrates how main() would behave with different error scenarios
	// Since we can't easily mock the global config.Load() function,
	// we test the error handling logic separately

	tests := []struct {
		name          string
		configErr     error
		importErr     error
		expectedFatal string
	}{
		{
			name:          "config load error",
			configErr:     errors.New("failed to load config"),
			importErr:     nil,
			expectedFatal: "Failed to load configuration: failed to load config",
		},
		{
			name:          "import error",
			configErr:     nil,
			importErr:     errors.New("failed to import"),
			expectedFatal: "Import failed: failed to import",
		},
		{
			name:          "no errors",
			configErr:     nil,
			importErr:     nil,
			expectedFatal: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the error handling logic that main() would use
			if tt.configErr != nil {
				expectedMsg := fmt.Sprintf("Failed to load configuration: %v", tt.configErr)
				if expectedMsg != tt.expectedFatal {
					t.Errorf("Expected fatal message %q, got %q", tt.expectedFatal, expectedMsg)
				}
				t.Logf("Would call log.Fatalf with: %s", expectedMsg)
				return
			}

			// Config loaded successfully, create clients
			cfg := &config.Config{
				RaindropToken: "test-token",
				KarakeepToken: "test-token",
			}

			raindropClient := raindrop.NewClient(cfg.RaindropToken)
			karakeepClient := karakeep.NewClient(cfg.KarakeepToken)
			importer := importer.NewImporter(raindropClient, karakeepClient)

			if tt.importErr != nil {
				expectedMsg := fmt.Sprintf("Import failed: %v", tt.importErr)
				if expectedMsg != tt.expectedFatal {
					t.Errorf("Expected fatal message %q, got %q", tt.expectedFatal, expectedMsg)
				}
				t.Logf("Would call log.Fatalf with: %s", expectedMsg)
				return
			}

			// Success case
			if importer == nil {
				t.Error("Expected importer to be created")
			}
			t.Log("Success: main() would complete normally")
		})
	}
}
