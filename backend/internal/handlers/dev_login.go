package handlers

import (
	"context"
	"encoding/json"
	"family-potluck/backend/internal/models"
	"net/http"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func (s *Server) DevLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	collection := s.DB.GetCollection("families")
	var family models.Family
	err := collection.FindOne(ctx, bson.M{"email": req.Email}).Decode(&family)

	if err == mongo.ErrNoDocuments {
		family = models.Family{
			ID:      primitive.NewObjectID(),
			Name:    req.Name,
			Email:   req.Email,
			Picture: "https://via.placeholder.com/150",
		}
		_, err = collection.InsertOne(ctx, family)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(family)
}
