package testutil

import (
	"log"
	"os"
)

// CleanupHelper provides utilities for test cleanup
type CleanupHelper struct{}

// NewCleanupHelper creates a new cleanup helper
func NewCleanupHelper() *CleanupHelper {
	return &CleanupHelper{}
}

// ShouldSkipCleanup checks if cleanup should be skipped based on environment variable
func (h *CleanupHelper) ShouldSkipCleanup() bool {
	skipCleanup := os.Getenv("SKIP_CLEANUP") != ""
	if skipCleanup {
		log.Println("SKIP_CLEANUP is set, cleanup will be skipped")
	}
	return skipCleanup
}

// LogCleanupStart logs the start of cleanup
func (h *CleanupHelper) LogCleanupStart() {
	log.Println("Starting test data cleanup...")
}

// LogCleanupComplete logs the completion of cleanup
func (h *CleanupHelper) LogCleanupComplete() {
	log.Println("Test data cleanup completed")
}

// LogCleanupSkipped logs that cleanup was skipped
func (h *CleanupHelper) LogCleanupSkipped() {
	log.Println("Skipping cleanup due to SKIP_CLEANUP environment variable")
}

/*
Example usage in integration tests:

```go
func setupCleanup(t *testing.T, client *karakeep.Client, testPrefix string) {
	helper := testutil.NewCleanupHelper()
	if helper.ShouldSkipCleanup() {
		helper.LogCleanupSkipped()
		return
	}
	
	t.Cleanup(func() {
		helper.LogCleanupStart()
		
		// Clean up bookmarks
		if err := cleanupBookmarks(client, testPrefix); err != nil {
			log.Printf("Failed to cleanup bookmarks: %v", err)
		}
		
		// Clean up lists
		if err := cleanupLists(client, testPrefix); err != nil {
			log.Printf("Failed to cleanup lists: %v", err)
		}
		
		helper.LogCleanupComplete()
	})
}
```
*/