//go:build !integration

package importer

import (
	"strings"

	"github.com/ashebanow/rainbridge/internal/karakeep"
	"github.com/ashebanow/rainbridge/internal/raindrop"
)

// Test fixture generators for edge case testing

// generateLongString creates a string of specified length
func generateLongString(length int) string {
	return strings.Repeat("a", length)
}

// generateUnicodeString creates a string with various Unicode characters
func generateUnicodeString() string {
	return "Hello 世界 🌍 مرحبا мир ñoño"
}

// Test fixtures for Raindrop data

// generateEmptyRaindrop creates a raindrop with minimal valid data
func generateEmptyRaindrop() raindrop.Raindrop {
	return raindrop.Raindrop{
		ID:    1,
		Link:  "https://example.com",
		Title: "",
	}
}

// generateRaindropWithSpecialChars creates a raindrop with special characters
func generateRaindropWithSpecialChars() raindrop.Raindrop {
	return raindrop.Raindrop{
		ID:      2,
		Title:   `Special <>&"' chars \ / | ? * : ; , . ! @ # $ % ^ & ( ) [ ] { }`,
		Excerpt: "Description with\nnewlines\tand\rspecial\bchars",
		Link:    "https://example.com/path?query=value&special=<>&",
		Tags:    []string{"tag-with-dash", "tag_with_underscore", "tag.with.dots"},
	}
}

// generateRaindropWithUnicode creates a raindrop with Unicode content
func generateRaindropWithUnicode() raindrop.Raindrop {
	return raindrop.Raindrop{
		ID:      3,
		Title:   generateUnicodeString(),
		Excerpt: "Unicode description: 🎉 emoji, 中文 Chinese, العربية Arabic, Русский Russian",
		Link:    "https://example.com/unicode/世界",
		Tags:    []string{"emoji-🏷️", "中文标签", "тег"},
	}
}

// generateRaindropWithLongContent creates a raindrop with very long fields
func generateRaindropWithLongContent() raindrop.Raindrop {
	return raindrop.Raindrop{
		ID:      4,
		Title:   generateLongString(500), // Very long title
		Excerpt: generateLongString(2000), // Very long description
		Link:    "https://example.com/" + generateLongString(1900), // URL near 2000 char limit
		Tags:    []string{generateLongString(100), generateLongString(100)},
	}
}

// generateRaindropWithInvalidURL creates a raindrop with malformed URL
func generateRaindropWithInvalidURL() raindrop.Raindrop {
	return raindrop.Raindrop{
		ID:    5,
		Title: "Invalid URL Bookmark",
		Link:  "not-a-valid-url",
	}
}

// generateLargeRaindropSet creates a large number of raindrops for performance testing
func generateLargeRaindropSet(count int) []raindrop.Raindrop {
	raindrops := make([]raindrop.Raindrop, count)
	for i := 0; i < count; i++ {
		raindrops[i] = raindrop.Raindrop{
			ID:      int64(i + 1000),
			Title:   "Bookmark " + string(rune(i)),
			Excerpt: "Description for bookmark number " + string(rune(i)),
			Link:    "https://example.com/bookmark/" + string(rune(i)),
			Tags:    []string{"tag1", "tag2", "bulk-import"},
		}
	}
	return raindrops
}

// Test fixtures for Collection data

// generateEmptyCollection creates a collection with minimal data
func generateEmptyCollection() raindrop.Collection {
	return raindrop.Collection{
		ID:    100,
		Title: "",
	}
}

// generateCollectionWithSpecialChars creates a collection with special characters
func generateCollectionWithSpecialChars() raindrop.Collection {
	return raindrop.Collection{
		ID:    101,
		Title: `Special <>&"' Collection \ / | ? * : ; , . ! @ # $ % ^ & ( ) [ ] { }`,
	}
}

// generateCollectionWithUnicode creates a collection with Unicode name
func generateCollectionWithUnicode() raindrop.Collection {
	return raindrop.Collection{
		ID:    102,
		Title: generateUnicodeString() + " Collection",
	}
}

// generateDuplicateCollections creates collections with identical names
func generateDuplicateCollections() []raindrop.Collection {
	return []raindrop.Collection{
		{ID: 200, Title: "Duplicate Name"},
		{ID: 201, Title: "Duplicate Name"},
		{ID: 202, Title: "Duplicate Name"},
	}
}

// Test fixtures for Karakeep data validation

// validateBookmarkForKarakeep checks if a bookmark is valid for Karakeep
func validateBookmarkForKarakeep(b *karakeep.Bookmark) []string {
	var errors []string
	
	if b.URL == "" {
		errors = append(errors, "URL is required")
	}
	
	if b.Title == "" {
		errors = append(errors, "Title is required")
	}
	
	if len(b.URL) > 2000 {
		errors = append(errors, "URL exceeds 2000 character limit")
	}
	
	return errors
}

// sanitizeBookmarkForKarakeep cleans up bookmark data for Karakeep
func sanitizeBookmarkForKarakeep(b *karakeep.Bookmark) *karakeep.Bookmark {
	sanitized := *b
	
	// Trim whitespace
	sanitized.URL = strings.TrimSpace(sanitized.URL)
	sanitized.Title = strings.TrimSpace(sanitized.Title)
	sanitized.Description = strings.TrimSpace(sanitized.Description)
	
	// Ensure title is not empty
	if sanitized.Title == "" {
		sanitized.Title = "Untitled"
	}
	
	// Truncate long URLs
	if len(sanitized.URL) > 2000 {
		sanitized.URL = sanitized.URL[:2000]
	}
	
	// Clean up tags
	cleanTags := make([]string, 0, len(sanitized.Tags))
	for _, tag := range sanitized.Tags {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			cleanTags = append(cleanTags, tag)
		}
	}
	sanitized.Tags = cleanTags
	
	return &sanitized
}