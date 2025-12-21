package handlers

import (
	"context"
	"encoding/json"
	"family-potluck/backend/internal/models"
	"net/http"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (s *Server) RSVPEvent(w http.ResponseWriter, r *http.Request) {
	var rsvp models.RSVP
	if err := json.NewDecoder(r.Body).Decode(&rsvp); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := s.DB.UpsertRSVP(context.Background(), &rsvp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rsvp.ID = id

	// Fetch Family Name for broadcast
	family, err := s.DB.GetFamilyByID(context.Background(), rsvp.FamilyID)
	if err == nil {
		rsvp.FamilyName = family.Name
	}

	// Broadcast update
	msg := map[string]interface{}{
		"type": "rsvp_updated",
		"data": rsvp,
	}
	msgBytes, _ := json.Marshal(msg)
	s.Hub.Broadcast(msgBytes)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(rsvp)
}

func (s *Server) GetRSVPs(w http.ResponseWriter, r *http.Request) {
	eventIDStr := r.URL.Query().Get("event_id")
	if eventIDStr == "" {
		http.Error(w, "event_id is required", http.StatusBadRequest)
		return
	}

	eventID, err := primitive.ObjectIDFromHex(eventIDStr)
	if err != nil {
		http.Error(w, "invalid event_id", http.StatusBadRequest)
		return
	}

	rsvps, err := s.DB.GetRSVPsByEventID(context.Background(), eventID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Populate Family details
	for i := range rsvps {
		family, err := s.DB.GetFamilyByID(context.Background(), rsvps[i].FamilyID)
		if err == nil {
			rsvps[i].FamilyName = family.Name
			rsvps[i].FamilyPicture = family.Picture
			rsvps[i].DietaryPreferences = family.DietaryPreferences
		}
	}

	json.NewEncoder(w).Encode(rsvps)
}
