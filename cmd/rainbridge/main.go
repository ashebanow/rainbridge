package main

import (
	"fmt"
	"log"

	"github.com/thiswillbeyourgithub/rainbridge/internal/config"
	"github.com/thiswillbeyourgithub/rainbridge/internal/karakeep"
	"github.com/thiswillbeyourgithub/rainbridge/internal/raindrop"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	raindropClient := raindrop.NewClient(cfg.RaindropToken)
	karakeepClient := karakeep.NewClient(cfg.KarakeepToken)

	raindrops, err := raindropClient.GetRaindrops()
	if err != nil {
		log.Fatalf("Failed to get raindrops: %v", err)
	}

	fmt.Printf("Fetched %d raindrops from Raindrop.io\n", len(raindrops))

	// Now, let's try to create a bookmark in Karakeep
	if len(raindrops) > 0 {
		firstRaindrop := raindrops[0]
		bookmark := &karakeep.Bookmark{
			URL:         firstRaindrop.Link,
			Title:       firstRaindrop.Title,
			Description: firstRaindrop.Excerpt,
			Tags:        firstRaindrop.Tags,
		}

		if err := karakeepClient.CreateBookmark(bookmark); err != nil {
			log.Fatalf("Failed to create bookmark in Karakeep: %v", err)
		}

		fmt.Println("Successfully created a bookmark in Karakeep!")
	}
}
