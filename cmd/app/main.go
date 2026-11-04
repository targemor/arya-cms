package main

import (
	"log"

	"github.com/lucimux/arya/config"
	"github.com/lucimux/arya/internal/app"
)

func main() {
	// Configuration
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Config error: %s", err)
	}

	// Run
	app.Run(cfg)
}
