package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"messenger/internal/app"
)

func main() {
	// Health check flag for Docker healthcheck in distroless containers
	if len(os.Args) > 1 && os.Args[1] == "-health" {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}

		client := &http.Client{Timeout: 3 * time.Second}
		resp, err := client.Get(fmt.Sprintf("http://localhost:%s/health", port))
		if err != nil {
			os.Exit(1)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			os.Exit(0)
		}
		os.Exit(1)
	}

	if err := app.Run(); err != nil {
		os.Exit(1)
	}
}
