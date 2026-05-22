package main

import (
	"fmt"
	"log"

	"gapi-server/internal/config"
)

func main() {
	// Initialize app via wire
	// engine, cleanup, err := WireApp()
	// if err != nil {
	// 	log.Fatalf("failed to initialize app: %v", err)
	// }
	// defer cleanup()

	// Placeholder: load config for server address until wire is generated
	v := config.NewViper()
	cfg := config.NewConfig(v)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("starting server on %s", addr)

	// After wire generation, use:
	// addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	// if err := engine.Run(addr); err != nil {
	// 	log.Fatalf("server failed: %v", err)
	// }

	log.Printf("wire not yet generated — run `wire ./cmd/server/` first, then uncomment WireApp call")
}
