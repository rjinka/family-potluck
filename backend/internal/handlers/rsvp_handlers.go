package handlers

import (
	"context"
	"encoding/json"
	"family-potluck/backend/internal/models"
	"net/http"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (s *Server) RSVPEvent(w http.ResponseWriter, r *http.Request) {
	var rsvp models.RSVP
	if err := json.NewDecoder(r.Body).Decode(&rsvp); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	collection := s.DB.GetCollection("rsvps")

	// Upsert RSVP
	filter := bson.M{
		"event_id":  rsvp.EventID,
		"family_id": rsvp.FamilyID,
	}
	update := bson.M{
		"$set": bson.M{
			"status":     rsvp.Status,
			"count":      rsvp.Count,
			"kids_count": rsvp.KidsCount,
		},
	}
	opts := options.Update().SetUpsert(true)

	result, err := collection.UpdateOne(context.Background(), filter, update, opts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// If upserted, we need to get the ID. If updated, we keep existing ID (not critical for frontend usually)
	// For simplicity, we can just return the input RSVP with success status
	if result.UpsertedID != nil {
		rsvp.ID = result.UpsertedID.(primitive.ObjectID)
	}

	// Fetch Family Name for broadcast
	familiesCollection := s.DB.GetCollection("families")
	var family models.Family
	err = familiesCollection.FindOne(context.Background(), bson.M{"_id": rsvp.FamilyID}).Decode(&family)
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

	collection := s.DB.GetCollection("rsvps")
	cursor, err := collection.Find(context.Background(), bson.M{"event_id": eventID})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.Background())

	var rsvps []models.RSVP
	if err = cursor.All(context.Background(), &rsvps); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Populate Family details
	familiesCollection := s.DB.GetCollection("families")
	for i := range rsvps {
		var family models.Family
		err := familiesCollection.FindOne(context.Background(), bson.M{"_id": rsvps[i].FamilyID}).Decode(&family)
		if err == nil {
			rsvps[i].FamilyName = family.Name
			rsvps[i].FamilyPicture = family.Picture
			rsvps[i].DietaryPreferences = family.DietaryPreferences
		}
	}

	json.NewEncoder(w).Encode(rsvps)
}
