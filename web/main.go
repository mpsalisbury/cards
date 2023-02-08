// Sample run-helloworld is a minimal Cloud Run service.
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	log.Print("starting server...")
	http.HandleFunc("/", root_handler)
	http.HandleFunc("/api", api_handler)

	// Determine port for HTTP service.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("defaulting to port %s", port)
	}

	// Start HTTP server.
	log.Printf("listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func root_handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to the Cards server.\n")
	fmt.Fprintf(w, "Connect your client to the api endpoint.\n")
}

func api_handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "API endpoint\n")
}
