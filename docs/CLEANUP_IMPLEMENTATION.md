# Test Cleanup Implementation

This document describes the cleanup functionality implemented for integration tests in the RainBridge project.

## Overview

The cleanup functionality ensures that test data created during integration tests is properly cleaned up after tests complete, preventing test pollution and maintaining clean test environments.

## Implementation Details

### 1. Enhanced Karakeep Client

Added new methods to the Karakeep client (`internal/karakeep/karakeep.go`):

- `GetAllBookmarks()` - Fetches all bookmarks from Karakeep
- `GetAllLists()` - Fetches all lists from Karakeep  
- `DeleteBookmark(bookmarkID string)` - Deletes a specific bookmark
- `DeleteList(listID string)` - Deletes a specific list

### 2. Test Utilities

Created `internal/testutil/cleanup.go` with a `CleanupHelper` that provides:

- `ShouldSkipCleanup()` - Checks if cleanup should be skipped via `SKIP_CLEANUP` env var
- Logging utilities for cleanup operations
- Example usage patterns

### 3. Integration Test Updates

Updated all integration tests with cleanup functionality:

#### Files Modified:
- `/Users/ashebanow/Development/tools/rainbridge/internal/karakeep/integration_test.go`
- `/Users/ashebanow/Development/tools/rainbridge/internal/importer/karakeep_integration_test.go`

#### Cleanup Pattern:
```go
// setupCleanup sets up test data cleanup
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

## Features

### 1. Automatic Cleanup
- Uses `t.Cleanup()` to ensure cleanup runs even if tests fail
- Cleanup is registered at the start of each test
- Runs after all test assertions complete

### 2. Test Prefix Filtering
- Only cleans up data with specific prefixes (e.g., `[Test]`)
- Prevents accidental deletion of production data
- Ensures test isolation

### 3. Configurable Cleanup
- `SKIP_CLEANUP` environment variable allows skipping cleanup
- Useful for debugging test failures
- Provides clear logging when cleanup is skipped

### 4. Comprehensive Logging
- Logs all cleanup operations
- Shows counts of deleted items
- Logs failures with error messages
- Provides clear start/end markers

### 5. Error Handling
- Cleanup continues even if some operations fail
- Individual failures are logged but don't stop the process
- Non-zero exit codes are handled gracefully

## Usage Examples

### Running Tests with Cleanup (Default)
```bash
go test -tags=integration ./internal/karakeep
```

### Running Tests without Cleanup (Debugging)
```bash
SKIP_CLEANUP=1 go test -tags=integration ./internal/karakeep
```

### Test Data Conventions
All test data should use the `[Test]` prefix:

```go
// Bookmarks
bookmark := &karakeep.Bookmark{
    URL:   "https://example.com/test",
    Title: "[Test] My Test Bookmark",  // ← Required prefix
}

// Lists
list := &karakeep.List{
    Name: "[Test] My Test List",  // ← Required prefix
}
```

## Test Data Created

The integration tests create the following test data that will be cleaned up:

### Karakeep Integration Test (`TestIntegrationCreateBookmark`)
- Creates bookmarks with title: `[Test] Integration Test Bookmark`
- Tags: `["test"]`
- URL: `https://example.com/integration-test`

### Karakeep Integration Test (`TestIntegrationCreateList`)
- Creates lists with name: `[Test] Integration Test List`

### Importer Integration Test (`TestKarakeepIntegration`)
- Creates lists with name: `[Test] Mocked Collection`
- Creates bookmarks with title: `[Test] Mocked Bookmark`
- URL: `https://example.com/mock`

## Safety Features

### 1. Prefix-Based Filtering
Only items with the `[Test]` prefix are deleted, ensuring:
- Production data is never accidentally deleted
- Other users' test data is not affected
- Clear separation between test and real data

### 2. API-Based Cleanup
- Uses the same API endpoints as the application
- Respects API rate limits and error handling
- Follows the same authentication patterns

### 3. Graceful Failure Handling
- Individual cleanup failures don't crash the test suite
- Detailed error logging for debugging
- Continue-on-error approach for robustness

## Files Modified

### New Files Created:
- `/Users/ashebanow/Development/tools/rainbridge/internal/testutil/cleanup.go`
- `/Users/ashebanow/Development/tools/rainbridge/internal/testutil/README.md`
- `/Users/ashebanow/Development/tools/rainbridge/internal/testutil/cleanup_demo_test.go`

### Existing Files Modified:
- `/Users/ashebanow/Development/tools/rainbridge/internal/karakeep/karakeep.go` - Added CRUD methods
- `/Users/ashebanow/Development/tools/rainbridge/internal/karakeep/integration_test.go` - Added cleanup
- `/Users/ashebanow/Development/tools/rainbridge/internal/importer/karakeep_integration_test.go` - Added cleanup

## Testing the Implementation

The cleanup functionality has been tested with:

1. **Compilation Tests**: All integration tests compile successfully
2. **Cleanup Execution**: Cleanup runs automatically after tests
3. **Skip Functionality**: `SKIP_CLEANUP` environment variable works correctly
4. **Error Handling**: Cleanup continues even when API calls fail
5. **Logging**: All cleanup operations are properly logged

## Benefits

1. **Clean Test Environment**: No test data pollution between runs
2. **Debugging Support**: Can skip cleanup to inspect test failures
3. **Safety**: Prefix-based filtering prevents accidental data deletion
4. **Maintainability**: Consistent cleanup patterns across all tests
5. **Observability**: Comprehensive logging of all cleanup operations