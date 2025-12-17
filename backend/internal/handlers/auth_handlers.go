package handlers

import (
	"context"
	"encoding/json"
	"family-potluck/backend/internal/models"
	"net/http"
	"os"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/api/idtoken"
)

func (s *Server) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDToken string `json:"id_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	// Ideally, validate the audience (your Client ID)
	payload, err := idtoken.Validate(ctx, req.IDToken, os.Getenv("GOOGLE_CLIENT_ID"))
	if err != nil {
		http.Error(w, "Invalid token: "+err.Error(), http.StatusUnauthorized)
		return
	}

	email := payload.Claims["email"].(string)
	name := payload.Claims["name"].(string)
	picture := payload.Claims["picture"].(string)
	sub := payload.Subject // Google ID

	collection := s.DB.GetCollection("families")
	var family models.Family
	err = collection.FindOne(ctx, bson.M{"email": email}).Decode(&family)

	if err == mongo.ErrNoDocuments {
		// Create new user
		family = models.Family{
			ID:       primitive.NewObjectID(),
			Name:     name,
			Email:    email,
			GoogleID: sub,
			Picture:  picture,
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
	} else {
		// Update existing user info if needed
		update := bson.M{
			"$set": bson.M{
				"name":      name,
				"picture":   picture,
				"google_id": sub,
			},
		}
		_, err = collection.UpdateOne(ctx, bson.M{"_id": family.ID}, update)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Create a session or JWT here if needed. For now, we'll just return the family object.
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(family)
}

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
