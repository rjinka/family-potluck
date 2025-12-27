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
		Name        string    `json:"name"`
		Date        time.Time `json:"date"`
		Location    string    `json:"location"`
		Description string    `json:"description"`
		Recurrence  string    `json:"recurrence"`
		Type        string    `json:"type"`
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

	// Verify permissions: Host, Host Household Member, or Group Admin
	isAuthorized := false
	if updates.UserID == event.HostID.Hex() {
		isAuthorized = true
	} else {
		// Check if user is in the same household as the host
		userOID, _ := primitive.ObjectIDFromHex(updates.UserID)
		var userMember models.FamilyMember
		var hostMember models.FamilyMember

		familyColl := s.DB.GetCollection("families")
		errUser := familyColl.FindOne(context.Background(), bson.M{"_id": userOID}).Decode(&userMember)
		errHost := familyColl.FindOne(context.Background(), bson.M{"_id": event.HostID}).Decode(&hostMember)

		if errUser == nil && errHost == nil && userMember.HouseholdID != nil && hostMember.HouseholdID != nil && *userMember.HouseholdID == *hostMember.HouseholdID {
			isAuthorized = true
		} else {
			// Check if user is group admin
			groupColl := s.DB.GetCollection("groups")
			var group models.Group
			errGroup := groupColl.FindOne(context.Background(), bson.M{"_id": event.GroupID}).Decode(&group)
			if errGroup == nil && isAdmin(group.AdminIDs, userOID) {
				isAuthorized = true
			}
		}
	}

	if !isAuthorized {
		http.Error(w, "You do not have permission to edit this event", http.StatusForbidden)
		return
	}

	updateFields := bson.M{}
	if updates.Name != "" {
		updateFields["name"] = updates.Name
	}
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
	// Recurrence can be cleared (set to empty)
	updateFields["recurrence"] = updates.Recurrence
	if updates.Recurrence != "" && event.RecurrenceID.IsZero() {
		updateFields["recurrence_id"] = primitive.NewObjectID()
	}
	if updates.Type != "" {
		updateFields["type"] = updates.Type
	}

	if len(updateFields) > 0 {
		_, err = collection.UpdateOne(context.Background(), bson.M{"_id": id}, bson.M{"$set": updateFields})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Update local event object for response/broadcast
		if val, ok := updateFields["name"]; ok {
			event.Name = val.(string)
		}
		if val, ok := updateFields["date"]; ok {
			event.Date = val.(time.Time)
		}
		if val, ok := updateFields["location"]; ok {
			event.Location = val.(string)
		}
		if val, ok := updateFields["description"]; ok {
			event.Description = val.(string)
		}
		if val, ok := updateFields["recurrence"]; ok {
			event.Recurrence = val.(string)
		}
		if val, ok := updateFields["recurrence_id"]; ok {
			event.RecurrenceID = val.(primitive.ObjectID)
		}
		if val, ok := updateFields["type"]; ok {
			event.Type = val.(string)
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
