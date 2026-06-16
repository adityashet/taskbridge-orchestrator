package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"taskbridge/internal/api"
	"taskbridge/internal/store"
)

func main() {
	// Keep the original command line flags
	addr := flag.String("addr", ":8080", "server listen address")
	flag.Parse()

	// 1. Initialize the thread-safe MemoryStore we built earlier
	memStore := store.NewMemoryStore()

	// 2. Instantiate our API handler package and pass our memory store into it
	apiHandler := api.NewHandler(memStore)

	// 3. Create a clean routing multiplexer
	mux := http.NewServeMux()

	// 4. Register all the custom core routes from internal/api
	apiHandler.RegisterRoutes(mux)

	// Keep the default base health check endpoint intact
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","service":"taskbridge-server"}`))
	})

	fmt.Printf("TaskBridge server listening on %s\n", *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
}
