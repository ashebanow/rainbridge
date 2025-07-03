package main

import (
	"log"

	"github.com/ashebanow/rainbridge/internal/config"
	"github.com/ashebanow/rainbridge/internal/importer"
	"github.com/ashebanow/rainbridge/internal/karakeep"
	"github.com/ashebanow/rainbridge/internal/raindrop"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	raindropClient := raindrop.NewClient(cfg.RaindropToken)
	karakeepClient := karakeep.NewClient(cfg.KarakeepToken)

	importer := importer.NewImporter(raindropClient, karakeepClient)

	if err := importer.RunImport(); err != nil {
		log.Fatalf("Import failed: %v", err)
	}
}
