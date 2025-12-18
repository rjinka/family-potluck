package handlers

import (
	"context"
	"family-potluck/backend/internal/database"
	"family-potluck/backend/internal/websocket"
	"net/http"

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
