package handlers

import (
	"context"
	"encoding/json"
	"family-potluck/backend/internal/models"
	"net/http"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (s *Server) CreateHousehold(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string             `json:"name"`
		FamilyID primitive.ObjectID `json:"family_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	household := models.Household{
		ID:        primitive.NewObjectID(),
		Name:      req.Name,
		MemberIDs: []primitive.ObjectID{req.FamilyID},
	}

	err := s.DB.CreateHousehold(context.Background(), &household)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update family with household ID
	err = s.DB.UpdateFamily(
		context.Background(),
		req.FamilyID,
		bson.M{"$set": bson.M{"household_id": household.ID}},
	)
	if err != nil {
		// Ideally rollback household creation
		http.Error(w, "Household created but failed to update family", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(household)
}

func (s *Server) JoinHousehold(w http.ResponseWriter, r *http.Request) {
	var req struct {
		HouseholdID primitive.ObjectID `json:"household_id"`
		FamilyID    primitive.ObjectID `json:"family_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Update Household
	err := s.DB.UpdateHousehold(
		context.Background(),
		req.HouseholdID,
		bson.M{"$addToSet": bson.M{"member_ids": req.FamilyID}},
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update Family
	err = s.DB.UpdateFamily(
		context.Background(),
		req.FamilyID,
		bson.M{"$set": bson.M{"household_id": req.HouseholdID}},
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) GetHousehold(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, "Invalid household id", http.StatusBadRequest)
		return
	}

	household, err := s.DB.GetHousehold(context.Background(), id)
	if err != nil {
		http.Error(w, "Household not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(household)
}

func (s *Server) AddMemberToHousehold(w http.ResponseWriter, r *http.Request) {
	var req struct {
		HouseholdID primitive.ObjectID `json:"household_id"`
		Email       string             `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Find family by email
	family, err := s.DB.GetFamilyByEmail(context.Background(), req.Email)
	if err != nil {
		http.Error(w, "User not found with that email", http.StatusNotFound)
		return
	}

	// Update Household
	err = s.DB.UpdateHousehold(
		context.Background(),
		req.HouseholdID,
		bson.M{"$addToSet": bson.M{"member_ids": family.ID}},
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update Family
	err = s.DB.UpdateFamily(
		context.Background(),
		family.ID,
		bson.M{"$set": bson.M{"household_id": req.HouseholdID}},
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
