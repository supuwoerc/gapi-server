package main

import (
	"flag"
	"fmt"
	"log"

	"gapi-server/internal/config"
)

func main() {
	cfgPath := flag.String("config", "configs/config.yaml", "path to config file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Initialize app via wire (uncomment after running `wire ./cmd/server/`)
	// engine, cleanup, err := InitializeApp(cfg)
	// if err != nil {
	// 	log.Fatalf("failed to initialize app: %v", err)
	// }
	// defer cleanup()

	// Start server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("starting server on %s", addr)

	// Placeholder: after wire generation, use:
	// if err := engine.Run(addr); err != nil {
	// 	log.Fatalf("server failed: %v", err)
	// }

	_ = cfg // remove after uncommenting above
	log.Printf("wire not yet generated — run `wire ./cmd/server/` first, then uncomment InitializeApp call")
}
