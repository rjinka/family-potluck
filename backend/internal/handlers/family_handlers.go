package handlers

import (
	"context"
	"encoding/json"
	"family-potluck/backend/internal/models"
	"net/http"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (s *Server) GetFamily(w http.ResponseWriter, r *http.Request) {
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

	collection := s.DB.GetCollection("families")
	var family models.Family
	err = collection.FindOne(context.Background(), bson.M{"_id": id}).Decode(&family)
	if err != nil {
		http.Error(w, "Family not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(family)
}
