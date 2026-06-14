package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// Server holds API dependencies
type Server struct {
	users []User
}

// User represents a user account
type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

// LoginRequest represents login credentials
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents login result
type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// NewServer creates a new API server
func NewServer() *Server {
	return &Server{
		users: []User{
			{ID: 1, Username: "alice", Email: "alice@example.com"},
			{ID: 2, Username: "bob", Email: "bob@example.com"},
		},
	}
}

// HealthHandler returns server health status
func (s *Server) HealthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"service":   "web-example-api",
	})
}

// UsersHandler returns list of users
func (s *Server) UsersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"users": s.users,
		"count": len(s.users),
	})
}

// LoginHandler handles user authentication
func (s *Server) LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Simple authentication (demo only - not secure)
	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "Username and password required")
		return
	}

	// Check if user exists
	userExists := false
	for _, u := range s.users {
		if u.Username == req.Username {
			userExists = true
			break
		}
	}

	if !userExists {
		writeError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Generate token (demo only - use JWT in production)
	token := generateDemoToken(req.Username)
	expiresAt := time.Now().Add(24 * time.Hour)

	writeJSON(w, http.StatusOK, LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
	})

	log.Printf("User logged in: %s", req.Username)
}

// Helper functions

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func generateDemoToken(username string) string {
	// Demo token (not secure - use JWT in production)
	return "demo_token_" + username + "_" + time.Now().Format("20060102150405")
}
