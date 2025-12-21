package handlers

import (
	"context"
	"encoding/json"
	"family-potluck/backend/internal/models"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (s *Server) SendChatMessage(w http.ResponseWriter, r *http.Request) {
	var msg models.ChatMessage
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validation: Check if user is a group member (not just a guest)
	// 1. Get Event to find GroupID
	event, err := s.DB.GetEvent(context.Background(), msg.EventID)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	// 2. Get Family to check Group Membership
	family, err := s.DB.GetFamilyByID(context.Background(), msg.FamilyID)
	if err != nil {
		http.Error(w, "Family not found", http.StatusNotFound)
		return
	}

	isMember := false
	for _, gid := range family.GroupIDs {
		if gid == event.GroupID {
			isMember = true
			break
		}
	}

	if !isMember {
		http.Error(w, "Chat is restricted to group members only", http.StatusForbidden)
		return
	}

	// Set metadata
	msg.ID = primitive.NewObjectID()
	msg.CreatedAt = time.Now()
	msg.FamilyName = family.Name // Ensure name is correct from DB

	// Save to DB
	err = s.DB.CreateChatMessage(context.Background(), &msg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Broadcast
	broadcastMsg := map[string]interface{}{
		"type": "new_chat_message",
		"data": msg,
	}
	msgBytes, _ := json.Marshal(broadcastMsg)
	s.Hub.Broadcast(msgBytes)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(msg)
}

func (s *Server) GetChatMessages(w http.ResponseWriter, r *http.Request) {
	eventIDStr := r.URL.Query().Get("event_id")
	if eventIDStr == "" {
		http.Error(w, "event_id is required", http.StatusBadRequest)
		return
	}

	eventID, err := primitive.ObjectIDFromHex(eventIDStr)
	if err != nil {
		http.Error(w, "Invalid event_id", http.StatusBadRequest)
		return
	}

	messages, err := s.DB.GetChatMessagesByEventID(context.Background(), eventID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(messages)
}
