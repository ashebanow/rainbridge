//go:build !integration

package importer

import (
	"strings"
	"testing"

	"github.com/ashebanow/rainbridge/internal/karakeep"
)

// TestValidateBookmarkForKarakeep tests bookmark validation logic
func TestValidateBookmarkForKarakeep(t *testing.T) {
	testCases := []struct {
		name           string
		bookmark       *karakeep.Bookmark
		expectedErrors []string
	}{
		{
			name: "Valid bookmark",
			bookmark: &karakeep.Bookmark{
				URL:   "https://example.com",
				Title: "Example",
			},
			expectedErrors: []string{},
		},
		{
			name: "Missing URL",
			bookmark: &karakeep.Bookmark{
				URL:   "",
				Title: "Example",
			},
			expectedErrors: []string{"URL is required"},
		},
		{
			name: "Missing title",
			bookmark: &karakeep.Bookmark{
				URL:   "https://example.com",
				Title: "",
			},
			expectedErrors: []string{"Title is required"},
		},
		{
			name: "Missing both URL and title",
			bookmark: &karakeep.Bookmark{
				URL:   "",
				Title: "",
			},
			expectedErrors: []string{"URL is required", "Title is required"},
		},
		{
			name: "URL exceeds limit",
			bookmark: &karakeep.Bookmark{
				URL:   "https://example.com/" + generateLongString(2000),
				Title: "Example",
			},
			expectedErrors: []string{"URL exceeds 2000 character limit"},
		},
		{
			name: "Whitespace only URL",
			bookmark: &karakeep.Bookmark{
				URL:   "   \t\n   ",
				Title: "Example",
			},
			expectedErrors: []string{"URL is required"}, // After trimming
		},
		{
			name: "Whitespace only title",
			bookmark: &karakeep.Bookmark{
				URL:   "https://example.com",
				Title: "   \t\n   ",
			},
			expectedErrors: []string{"Title is required"}, // After trimming
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Trim whitespace before validation (simulating real behavior)
			tc.bookmark.URL = strings.TrimSpace(tc.bookmark.URL)
			tc.bookmark.Title = strings.TrimSpace(tc.bookmark.Title)
			
			errors := validateBookmarkForKarakeep(tc.bookmark)
			
			if len(errors) != len(tc.expectedErrors) {
				t.Errorf("Expected %d errors, got %d: %v", len(tc.expectedErrors), len(errors), errors)
				return
			}
			
			for i, expectedError := range tc.expectedErrors {
				if errors[i] != expectedError {
					t.Errorf("Expected error %q, got %q", expectedError, errors[i])
				}
			}
		})
	}
}

// TestSanitizeBookmarkForKarakeep tests bookmark sanitization
func TestSanitizeBookmarkForKarakeep(t *testing.T) {
	testCases := []struct {
		name     string
		input    *karakeep.Bookmark
		expected *karakeep.Bookmark
	}{
		{
			name: "No sanitization needed",
			input: &karakeep.Bookmark{
				URL:         "https://example.com",
				Title:       "Example",
				Description: "Description",
				Tags:        []string{"tag1", "tag2"},
			},
			expected: &karakeep.Bookmark{
				URL:         "https://example.com",
				Title:       "Example",
				Description: "Description",
				Tags:        []string{"tag1", "tag2"},
			},
		},
		{
			name: "Trim whitespace",
			input: &karakeep.Bookmark{
				URL:         "  https://example.com  ",
				Title:       "\tExample\n",
				Description: "  Description  ",
				Tags:        []string{"  tag1  ", "\ttag2\n"},
			},
			expected: &karakeep.Bookmark{
				URL:         "https://example.com",
				Title:       "Example",
				Description: "Description",
				Tags:        []string{"tag1", "tag2"},
			},
		},
		{
			name: "Empty title becomes Untitled",
			input: &karakeep.Bookmark{
				URL:   "https://example.com",
				Title: "",
			},
			expected: &karakeep.Bookmark{
				URL:   "https://example.com",
				Title: "Untitled",
			},
		},
		{
			name: "Whitespace-only title becomes Untitled",
			input: &karakeep.Bookmark{
				URL:   "https://example.com",
				Title: "   \t\n   ",
			},
			expected: &karakeep.Bookmark{
				URL:   "https://example.com",
				Title: "Untitled",
			},
		},
		{
			name: "Truncate long URL",
			input: &karakeep.Bookmark{
				URL:   "https://example.com/" + generateLongString(2100),
				Title: "Example",
			},
			expected: &karakeep.Bookmark{
				URL:   "https://example.com/" + generateLongString(2000-20), // 20 chars for base URL
				Title: "Example",
			},
		},
		{
			name: "Remove empty tags",
			input: &karakeep.Bookmark{
				URL:   "https://example.com",
				Title: "Example",
				Tags:  []string{"tag1", "", "  ", "tag2", "\t\n"},
			},
			expected: &karakeep.Bookmark{
				URL:   "https://example.com",
				Title: "Example",
				Tags:  []string{"tag1", "tag2"},
			},
		},
		{
			name: "Handle nil tags",
			input: &karakeep.Bookmark{
				URL:   "https://example.com",
				Title: "Example",
				Tags:  nil,
			},
			expected: &karakeep.Bookmark{
				URL:   "https://example.com",
				Title: "Example",
				Tags:  []string{},
			},
		},
		{
			name: "Complex sanitization",
			input: &karakeep.Bookmark{
				URL:         "  https://example.com/" + generateLongString(2100) + "  ",
				Title:       "   ",
				Description: "\n\nDescription with\nextra whitespace\n\n",
				Tags:        []string{"", "tag1", "  ", "  tag2  ", ""},
			},
			expected: &karakeep.Bookmark{
				URL:         "https://example.com/" + generateLongString(2000-20),
				Title:       "Untitled",
				Description: "Description with\nextra whitespace",
				Tags:        []string{"tag1", "tag2"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sanitizeBookmarkForKarakeep(tc.input)
			
			if result.URL != tc.expected.URL {
				t.Errorf("URL: expected %q, got %q", tc.expected.URL, result.URL)
			}
			if result.Title != tc.expected.Title {
				t.Errorf("Title: expected %q, got %q", tc.expected.Title, result.Title)
			}
			if result.Description != tc.expected.Description {
				t.Errorf("Description: expected %q, got %q", tc.expected.Description, result.Description)
			}
			if len(result.Tags) != len(tc.expected.Tags) {
				t.Errorf("Tags: expected %d tags, got %d", len(tc.expected.Tags), len(result.Tags))
			} else {
				for i, tag := range result.Tags {
					if tag != tc.expected.Tags[i] {
						t.Errorf("Tag[%d]: expected %q, got %q", i, tc.expected.Tags[i], tag)
					}
				}
			}
		})
	}
}

// TestGenerateLongString tests the long string generator
func TestGenerateLongString(t *testing.T) {
	testCases := []struct {
		length int
	}{
		{0},
		{1},
		{100},
		{1000},
		{2000},
		{10000},
	}

	for _, tc := range testCases {
		t.Run(string(rune(tc.length))+" chars", func(t *testing.T) {
			result := generateLongString(tc.length)
			if len(result) != tc.length {
				t.Errorf("Expected string of length %d, got %d", tc.length, len(result))
			}
			if tc.length > 0 && !strings.HasPrefix(result, "a") {
				t.Error("Expected string to consist of 'a' characters")
			}
		})
	}
}

// TestGenerateUnicodeString tests Unicode string generation
func TestGenerateUnicodeString(t *testing.T) {
	result := generateUnicodeString()
	
	// Check for expected substrings
	expectedSubstrings := []string{
		"Hello",
		"世界",    // Chinese
		"🌍",     // Emoji
		"مرحبا",  // Arabic
		"мир",    // Russian
		"ñoño",   // Spanish with special chars
	}
	
	for _, expected := range expectedSubstrings {
		if !strings.Contains(result, expected) {
			t.Errorf("Expected Unicode string to contain %q", expected)
		}
	}
}

// TestFixtureGeneration tests all fixture generators
func TestFixtureGeneration(t *testing.T) {
	t.Run("Empty Raindrop", func(t *testing.T) {
		r := generateEmptyRaindrop()
		if r.ID == 0 {
			t.Error("Expected non-zero ID")
		}
		if r.Link == "" {
			t.Error("Expected non-empty link")
		}
		if r.Title != "" {
			t.Error("Expected empty title")
		}
	})

	t.Run("Raindrop with special chars", func(t *testing.T) {
		r := generateRaindropWithSpecialChars()
		if !strings.Contains(r.Title, "<>&") {
			t.Error("Expected title to contain special HTML characters")
		}
		if !strings.Contains(r.Excerpt, "\n") {
			t.Error("Expected excerpt to contain newline")
		}
	})

	t.Run("Raindrop with Unicode", func(t *testing.T) {
		r := generateRaindropWithUnicode()
		if !strings.Contains(r.Title, "世界") {
			t.Error("Expected title to contain Unicode characters")
		}
		if len(r.Tags) == 0 {
			t.Error("Expected tags with Unicode content")
		}
	})

	t.Run("Large raindrop set", func(t *testing.T) {
		set := generateLargeRaindropSet(100)
		if len(set) != 100 {
			t.Errorf("Expected 100 raindrops, got %d", len(set))
		}
		// Check uniqueness of IDs
		idMap := make(map[int64]bool)
		for _, r := range set {
			if idMap[r.ID] {
				t.Errorf("Duplicate ID found: %d", r.ID)
			}
			idMap[r.ID] = true
		}
	})

	t.Run("Collections", func(t *testing.T) {
		empty := generateEmptyCollection()
		if empty.Title != "" {
			t.Error("Expected empty collection title")
		}

		special := generateCollectionWithSpecialChars()
		if !strings.Contains(special.Title, "<>&") {
			t.Error("Expected special characters in collection title")
		}

		unicode := generateCollectionWithUnicode()
		if !strings.Contains(unicode.Title, "世界") {
			t.Error("Expected Unicode in collection title")
		}

		duplicates := generateDuplicateCollections()
		if len(duplicates) != 3 {
			t.Errorf("Expected 3 duplicate collections, got %d", len(duplicates))
		}
		for _, c := range duplicates {
			if c.Title != "Duplicate Name" {
				t.Error("Expected all collections to have duplicate name")
			}
		}
	})
}