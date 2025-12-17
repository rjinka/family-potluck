package handlers

import (
	"context"
	"encoding/json"
	"family-potluck/backend/internal/models"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (s *Server) SendChatMessage(w http.ResponseWriter, r *http.Request) {
	var msg models.ChatMessage
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validation: Check if user is a group member (not just a guest)
	// 1. Get Event to find GroupID
	eventsCollection := s.DB.GetCollection("events")
	var event models.Event
	err := eventsCollection.FindOne(context.Background(), bson.M{"_id": msg.EventID}).Decode(&event)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	// 2. Get Family to check Group Membership
	familiesCollection := s.DB.GetCollection("families")
	var family models.Family
	err = familiesCollection.FindOne(context.Background(), bson.M{"_id": msg.FamilyID}).Decode(&family)
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
	chatCollection := s.DB.GetCollection("chat_messages")
	_, err = chatCollection.InsertOne(context.Background(), msg)
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

	collection := s.DB.GetCollection("chat_messages")

	// Sort by CreatedAt ascending (oldest first)
	opts := options.Find().SetSort(bson.M{"created_at": 1})

	cursor, err := collection.Find(context.Background(), bson.M{"event_id": eventID}, opts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.Background())

	var messages []models.ChatMessage
	if err = cursor.All(context.Background(), &messages); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(messages)
}
