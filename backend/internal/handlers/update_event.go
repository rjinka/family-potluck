package handlers

import (
	"context"
	"encoding/json"
	"family-potluck/backend/internal/models"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (s *Server) UpdateEvent(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, "Invalid event id", http.StatusBadRequest)
		return
	}

	var updates struct {
		Date        time.Time `json:"date"`
		Location    string    `json:"location"`
		Description string    `json:"description"`
		UserID      string    `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	collection := s.DB.GetCollection("events")
	var event models.Event
	err = collection.FindOne(context.Background(), bson.M{"_id": id}).Decode(&event)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	// Verify host
	if updates.UserID != event.HostID.Hex() {
		http.Error(w, "Only the host can edit event details", http.StatusForbidden)
		return
	}

	updateFields := bson.M{}
	if !updates.Date.IsZero() {
		updateFields["date"] = updates.Date
	}
	if updates.Location != "" {
		updateFields["location"] = updates.Location
	}
	// Description can be empty, but we might want to allow clearing it?
	// For now, only update if not empty string. If they want to clear, they might send a space?
	// Or we can use a pointer to string to distinguish nil vs empty.
	// But let's keep it simple: if not empty, update.
	if updates.Description != "" {
		updateFields["description"] = updates.Description
	}

	if len(updateFields) > 0 {
		_, err = collection.UpdateOne(context.Background(), bson.M{"_id": id}, bson.M{"$set": updateFields})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Update local event object for response/broadcast
		if val, ok := updateFields["date"]; ok {
			event.Date = val.(time.Time)
		}
		if val, ok := updateFields["location"]; ok {
			event.Location = val.(string)
		}
		if val, ok := updateFields["description"]; ok {
			event.Description = val.(string)
		}

		// Broadcast update
		msg := map[string]interface{}{
			"type": "event_updated",
			"data": event,
		}
		msgBytes, _ := json.Marshal(msg)
		s.Hub.Broadcast(msgBytes)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(event)
}
