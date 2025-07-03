package main

import (
	"log"

	"github.com/thiswillbeyourgithub/rainbridge/internal/config"
	"github.com/thiswillbeyourgithub/rainbridge/internal/importer"
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

	importer := importer.NewImporter(raindropClient, karakeepClient)

	if err := importer.RunImport(); err != nil {
		log.Fatalf("Import failed: %v", err)
	}
}
