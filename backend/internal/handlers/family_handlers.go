package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (s *Server) GetFamilyMember(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "Missing id parameter", http.StatusBadRequest)
		return
	}

	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, "Invalid id format", http.StatusBadRequest)
		return
	}

	familyMember, err := s.DB.GetFamilyMemberByID(context.Background(), id)
	if err != nil {
		http.Error(w, "Family member not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(familyMember)
}

func (s *Server) UpdateFamilyMember(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	if idStr == "" {
		http.Error(w, "Missing id parameter", http.StatusBadRequest)
		return
	}

	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, "Invalid id format", http.StatusBadRequest)
		return
	}

	var updateData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Whitelist allowed fields to update
	allowedFields := map[string]bool{
		"dietary_preferences": true,
		"allergies":           true,
		"address":             true,
	}

	update := bson.M{}
	for k, v := range updateData {
		if allowedFields[k] {
			update[k] = v
		}
	}

	if len(update) == 0 {
		http.Error(w, "No valid fields to update", http.StatusBadRequest)
		return
	}

	err = s.DB.UpdateFamilyMember(context.Background(), id, bson.M{"$set": update})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
