package main

import (
	"log"
	"os"

	"github.com/supuwoerc/gapi-server/cmd/cli/commands/system"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
	AppEnv    = "dev"
)

func main() {
	system.Version = Version
	system.BuildTime = BuildTime

	if os.Getenv("APP_ENV") == "" {
		if err := os.Setenv("APP_ENV", AppEnv); err != nil {
			log.Fatalf("failed to set APP_ENV: %v", err)
		}
	}
	Execute()
}
