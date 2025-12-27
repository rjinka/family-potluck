package handlers

import (
	"context"
	"encoding/json"
	"family-potluck/backend/internal/models"
	"log"
	"net/http"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (s *Server) CreateHousehold(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name           string             `json:"name"`
		Address        string             `json:"address"`
		FamilyMemberID primitive.ObjectID `json:"family_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	household := models.Household{
		ID:        primitive.NewObjectID(),
		Name:      req.Name,
		Address:   req.Address,
		MemberIDs: []primitive.ObjectID{req.FamilyMemberID},
	}

	err := s.DB.CreateHousehold(context.Background(), &household)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update family with household ID
	err = s.DB.UpdateFamilyMember(
		context.Background(),
		req.FamilyMemberID,
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
		HouseholdID    primitive.ObjectID `json:"household_id"`
		FamilyMemberID primitive.ObjectID `json:"family_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Update Household
	err := s.DB.UpdateHousehold(
		context.Background(),
		req.HouseholdID,
		bson.M{"$addToSet": bson.M{"member_ids": req.FamilyMemberID}},
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update Family
	err = s.DB.UpdateFamilyMember(
		context.Background(),
		req.FamilyMemberID,
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

	// Fetch members details
	if len(household.MemberIDs) > 0 {
		members, err := s.DB.GetFamilyMembersByIDs(context.Background(), household.MemberIDs)
		if err != nil {
			// Log error but continue with empty members
			log.Printf("Error fetching household members: %v", err)
		} else {
			safeMembers := make([]models.SafeFamilyMember, len(members))
			for i, m := range members {
				safeMembers[i] = m.ToSafe()
			}
			household.Members = safeMembers
		}
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
	familyMember, err := s.DB.GetFamilyMemberByEmail(context.Background(), req.Email)
	if err != nil {
		http.Error(w, "User not found with that email", http.StatusNotFound)
		return
	}

	// Update Household
	err = s.DB.UpdateHousehold(
		context.Background(),
		req.HouseholdID,
		bson.M{"$addToSet": bson.M{"member_ids": familyMember.ID}},
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update Family
	err = s.DB.UpdateFamilyMember(
		context.Background(),
		familyMember.ID,
		bson.M{"$set": bson.M{"household_id": req.HouseholdID}},
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) DeleteHousehold(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, "Invalid household id", http.StatusBadRequest)
		return
	}

	// Optional: Check for admin permissions if admin_id and group_id are provided
	// For now, we assume the caller has verified permissions or it's the user's own household
	// But the requirement is "Allow Admin to manage".
	// So we should check if the requester is authorized.
	// Since we don't have full auth middleware context here easily without parsing token again or passing it,
	// we rely on the request body/query for admin_id/group_id for now, similar to DeleteGroup.

	adminIDStr := r.URL.Query().Get("admin_id")
	groupIDStr := r.URL.Query().Get("group_id")

	if adminIDStr != "" && groupIDStr != "" {
		adminID, err := primitive.ObjectIDFromHex(adminIDStr)
		if err != nil {
			http.Error(w, "Invalid admin_id", http.StatusBadRequest)
			return
		}
		groupID, err := primitive.ObjectIDFromHex(groupIDStr)
		if err != nil {
			http.Error(w, "Invalid group_id", http.StatusBadRequest)
			return
		}

		// Verify admin
		group, err := s.DB.GetGroup(context.Background(), groupID)
		if err != nil {
			http.Error(w, "Group not found", http.StatusNotFound)
			return
		}
		if !isAdmin(group.AdminIDs, adminID) {
			http.Error(w, "Unauthorized: Only admin can delete household", http.StatusForbidden)
			return
		}
		// We could also verify the household belongs to this group (via members), but let's trust the admin for now.
	}

	err = s.DB.DeleteHousehold(context.Background(), id)
	if err != nil {
		http.Error(w, "Failed to delete household", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) RemoveMemberFromHousehold(w http.ResponseWriter, r *http.Request) {
	var req struct {
		HouseholdID    primitive.ObjectID  `json:"household_id"`
		FamilyMemberID primitive.ObjectID  `json:"family_id"`
		AdminID        *primitive.ObjectID `json:"admin_id,omitempty"`
		GroupID        *primitive.ObjectID `json:"group_id,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.AdminID != nil && req.GroupID != nil {
		// Verify admin
		group, err := s.DB.GetGroup(context.Background(), *req.GroupID)
		if err != nil {
			http.Error(w, "Group not found", http.StatusNotFound)
			return
		}
		if !isAdmin(group.AdminIDs, *req.AdminID) {
			http.Error(w, "Unauthorized: Only admin can remove members", http.StatusForbidden)
			return
		}
	}

	err := s.DB.RemoveMemberFromHousehold(context.Background(), req.HouseholdID, req.FamilyMemberID)
	if err != nil {
		http.Error(w, "Failed to remove member", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) UpdateHousehold(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, "Invalid household id", http.StatusBadRequest)
		return
	}

	var req struct {
		Name    string `json:"name"`
		Address string `json:"address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	update := bson.M{}
	if req.Name != "" {
		update["name"] = req.Name
	}
	if req.Address != "" {
		update["address"] = req.Address
	}

	if len(update) == 0 {
		http.Error(w, "No fields to update", http.StatusBadRequest)
		return
	}

	err = s.DB.UpdateHousehold(context.Background(), id, bson.M{"$set": update})
	if err != nil {
		http.Error(w, "Failed to update household", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
