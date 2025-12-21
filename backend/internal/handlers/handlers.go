package handlers

import (
	"context"
	"encoding/json"
	"family-potluck/backend/internal/database"
	"family-potluck/backend/internal/websocket"
	"net/http"
	"os"
	"strings"

	"google.golang.org/api/idtoken"
)

type TokenValidator interface {
	Validate(ctx context.Context, idToken string, audience string) (*idtoken.Payload, error)
}

type RealTokenValidator struct{}

func (v *RealTokenValidator) Validate(ctx context.Context, idToken string, audience string) (*idtoken.Payload, error) {
	return idtoken.Validate(ctx, idToken, audience)
}

type Server struct {
	DB             database.Service
	Hub            *websocket.Hub
	TokenValidator TokenValidator
}

func NewServer(db database.Service, hub *websocket.Hub) *Server {
	return &Server{
		DB:             db,
		Hub:            hub,
		TokenValidator: &RealTokenValidator{},
	}
}

func (s *Server) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (s *Server) GetVersion(w http.ResponseWriter, r *http.Request) {
	version, err := os.ReadFile("VERSION")
	if err != nil {
		http.Error(w, "Could not read version", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"version": strings.TrimSpace(string(version)),
	})
}
