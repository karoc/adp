package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/adp/web-example/backend/api"
)

func main() {
	// Initialize API server
	server := api.NewServer()

	// Setup routes
	http.HandleFunc("/api/health", server.HealthHandler)
	http.HandleFunc("/api/users", server.UsersHandler)
	http.HandleFunc("/api/auth/login", server.LoginHandler)

	// CORS middleware for development
	handler := corsMiddleware(http.DefaultServeMux)

	port := ":8080"
	log.Printf("Server starting on http://localhost%s", port)
	log.Printf("API endpoints:")
	log.Printf("  GET  /api/health")
	log.Printf("  GET  /api/users")
	log.Printf("  POST /api/auth/login")

	if err := http.ListenAndServe(port, handler); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// corsMiddleware adds CORS headers for development
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// writeJSON writes JSON response
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON: %v", err)
	}
}

// writeError writes error response
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
