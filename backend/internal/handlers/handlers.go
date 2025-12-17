package handlers

import (
	"family-potluck/backend/internal/database"
	"family-potluck/backend/internal/websocket"
	"net/http"
)

type Server struct {
	DB  database.Service
	Hub *websocket.Hub
}

func NewServer(db database.Service, hub *websocket.Hub) *Server {
	return &Server{
		DB:  db,
		Hub: hub,
	}
}

func (s *Server) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
