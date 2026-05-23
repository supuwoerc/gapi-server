package main

import "log"

func main() {
	application, err := WireApp()
	if err != nil {
		log.Fatalf("failed to initialize app: %v", err)
	}
	defer application.Close()
	application.Run()
}
