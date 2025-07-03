package importer

import (
	"fmt"
	"log"

	"github.com/ashebanow/rainbridge/internal/karakeep"
	"github.com/ashebanow/rainbridge/internal/raindrop"
)

// Importer holds the clients for the Raindrop and Karakeep APIs.
type Importer struct {
	RaindropClient *raindrop.Client
	KarakeepClient *karakeep.Client
}

// NewImporter creates a new Importer.
func NewImporter(raindropClient *raindrop.Client, karakeepClient *karakeep.Client) *Importer {
	return &Importer{
		RaindropClient: raindropClient,
		KarakeepClient: karakeepClient,
	}
}

// RunImport performs the full import process.
func (i *Importer) RunImport() error {
	// 1. Fetch collections from Raindrop.io
	fmt.Println("Fetching collections from Raindrop.io...")
	collections, err := i.RaindropClient.GetCollections()
	if err != nil {
		return fmt.Errorf("failed to get collections: %w", err)
	}
	fmt.Printf("Fetched %d collections.\n", len(collections))

	// 2. Create corresponding lists in Karakeep
	fmt.Println("Creating lists in Karakeep...")
	collectionMap := make(map[int64]string)
	for _, collection := range collections {
		list := &karakeep.List{Name: collection.Title}
		createdList, err := i.KarakeepClient.CreateList(list)
		if err != nil {
			log.Printf("Failed to create list '%s': %v", collection.Title, err)
			continue
		}
		collectionMap[collection.ID] = createdList.ID
		fmt.Printf("Created list: %s\n", createdList.Name)
	}

	// 3. Fetch bookmarks for each collection and import
	fmt.Println("\nImporting bookmarks...")
	for _, collection := range collections {
		fmt.Printf("\nFetching bookmarks for collection: %s\n", collection.Title)
		raindrops, err := i.RaindropClient.GetRaindropsByCollection(collection.ID)
		if err != nil {
			log.Printf("Failed to get raindrops for collection '%s': %v", collection.Title, err)
			continue
		}
		fmt.Printf("Found %d bookmarks in this collection.\n", len(raindrops))

		listID := collectionMap[collection.ID]

		for _, raindrop := range raindrops {
			bookmark := &karakeep.Bookmark{
				URL:         raindrop.Link,
				Title:       raindrop.Title,
				Description: raindrop.Excerpt,
				Tags:        raindrop.Tags,
			}

			createdBookmark, err := i.KarakeepClient.CreateBookmark(bookmark)
			if err != nil {
				log.Printf("Failed to create bookmark '%s': %v", raindrop.Title, err)
				continue
			}
			fmt.Printf("  - Created bookmark: %s\n", createdBookmark.Title)

			if err := i.KarakeepClient.AddBookmarkToList(createdBookmark.ID, listID); err != nil {
				log.Printf("Failed to add bookmark '%s' to list: %v", createdBookmark.Title, err)
			}
		}
	}

	fmt.Println("\nImport complete!")
	return nil
}
