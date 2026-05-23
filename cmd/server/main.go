package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
	AppEnv    = "dev"
	GinMode   = "debug"
)

func main() {
	if os.Getenv("APP_ENV") == "" {
		if err := os.Setenv("APP_ENV", AppEnv); err != nil {
			log.Fatalf("failed to set APP_ENV: %v", err)
		}
	}
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(GinMode)
	}

	log.Printf("gapi-server version=%s build_time=%s env=%s gin_mode=%s",
		Version, BuildTime, os.Getenv("APP_ENV"), gin.Mode())

	application, err := WireApp()
	if err != nil {
		log.Fatalf("failed to initialize app: %v", err)
	}
	defer application.Close()
	application.Run()
}
