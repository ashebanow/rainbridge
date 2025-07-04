# Test Utilities

This package provides utilities for testing, including cleanup functionality for integration tests.

## Test Data Cleanup

The `TestDataCleaner` provides automated cleanup of test data created during integration tests to prevent test pollution and ensure clean test environments.

### Usage

```go
import "github.com/ashebanow/rainbridge/internal/testutil"

func TestSomething(t *testing.T) {
    // Set up your clients
    karakeepClient := karakeep.NewClient(token)
    
    // Create cleanup with test prefix
    cleaner := testutil.NewTestDataCleaner(karakeepClient, "[Test]")
    cleaner.RegisterCleanup(t)
    
    // Your test code here - create bookmarks/lists with "[Test]" prefix
    // They will be automatically cleaned up after the test
}
```

### Features

- **Automatic cleanup**: Registers cleanup functions with `t.Cleanup()` to ensure cleanup runs even if tests fail
- **Test prefix filtering**: Only cleans up data with the specified prefix (e.g., "[Test]")
- **Configurable**: Skip cleanup by setting `SKIP_CLEANUP` environment variable
- **Logging**: Provides detailed logging of cleanup operations
- **Immediate cleanup**: Use `CleanupNow()` for debugging or manual cleanup

### Environment Variables

- `SKIP_CLEANUP`: If set to any value, cleanup operations will be skipped. Useful for debugging test failures.

### Test Prefixes

All test data should use consistent prefixes to ensure proper cleanup:
- Bookmarks: `[Test] Test Bookmark Title`
- Lists: `[Test] Test List Name`

The cleanup system will only remove items whose titles/names start with the specified prefix, ensuring production data is never accidentally deleted.

### Example

```go
// Create test data with proper prefix
bookmark := &karakeep.Bookmark{
    URL:   "https://example.com/test",
    Title: "[Test] My Test Bookmark",  // Prefix ensures cleanup
    Tags:  []string{"test"},
}

list := &karakeep.List{
    Name: "[Test] My Test List",  // Prefix ensures cleanup
}
```

### Running Tests with Cleanup

```bash
# Run integration tests with cleanup
go test -tags=integration ./internal/testutil

# Run integration tests without cleanup (for debugging)
SKIP_CLEANUP=1 go test -tags=integration ./internal/testutil
```