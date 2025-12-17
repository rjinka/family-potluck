package main

import (
	"family-potluck/backend/internal/database"
	"family-potluck/backend/internal/handlers"
	"family-potluck/backend/internal/websocket"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbService := database.New()
	defer dbService.Close()

	hub := websocket.NewHub()
	go hub.Run()

	server := handlers.NewServer(dbService, hub)

	mux := http.NewServeMux()

	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		hub.ServeWs(w, r)
	})

	mux.HandleFunc("POST /auth/google", server.GoogleLogin)
	mux.HandleFunc("POST /auth/dev", server.DevLogin)
	mux.HandleFunc("POST /groups", server.CreateGroup)
	mux.HandleFunc("POST /groups/join", server.JoinGroup)
	mux.HandleFunc("POST /groups/leave", server.LeaveGroup)
	mux.HandleFunc("POST /groups/join-by-code", server.JoinGroupByCode)
	mux.HandleFunc("GET /groups/code/{code}", server.GetGroupByCode)
	mux.HandleFunc("DELETE /groups/{id}", server.DeleteGroup)
	mux.HandleFunc("GET /groups/{id}", server.GetGroup)
	mux.HandleFunc("GET /groups", server.GetGroups)
	mux.HandleFunc("GET /family", server.GetFamily)
	mux.HandleFunc("POST /events", server.CreateEvent)
	mux.HandleFunc("POST /events/{id}/finish", server.FinishEvent)
	mux.HandleFunc("POST /events/{id}/skip", server.SkipEvent)
	mux.HandleFunc("DELETE /events/{id}", server.DeleteEvent)
	mux.HandleFunc("GET /events/{id}", server.GetEvent)
	mux.HandleFunc("PATCH /events/{id}", server.UpdateEvent)
	mux.HandleFunc("GET /events/stats/{id}", server.GetEventStats)
	mux.HandleFunc("GET /events", server.GetEvents)
	mux.HandleFunc("GET /events/user", server.GetUserEvents)
	mux.HandleFunc("POST /events/join-by-code", server.JoinEventByCode)
	mux.HandleFunc("GET /events/code/{code}", server.GetEventByCode)
	mux.HandleFunc("POST /rsvps", server.RSVPEvent)
	mux.HandleFunc("GET /rsvps", server.GetRSVPs)
	mux.HandleFunc("GET /groups/members", server.GetGroupMembers)
	mux.HandleFunc("POST /dishes", server.AddDish)
	mux.HandleFunc("GET /dishes", server.GetDishes)
	mux.HandleFunc("POST /dishes/{id}/pledge", server.PledgeDish)
	mux.HandleFunc("POST /dishes/{id}/unpledge", server.UnpledgeDish)
	mux.HandleFunc("DELETE /dishes/{id}", server.DeleteDish)
	mux.HandleFunc("POST /swaps", server.CreateSwapRequest)
	mux.HandleFunc("PATCH /swaps/{id}", server.UpdateSwapRequest)
	mux.HandleFunc("GET /swaps", server.GetSwapRequests)
	mux.HandleFunc("GET /health", server.HealthHandler)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      enableCORS(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	fmt.Printf("Server starting on port %s\n", port)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*") // For development
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
