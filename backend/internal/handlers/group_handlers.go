package handlers

import (
	"context"
	"encoding/json"
	"family-potluck/backend/internal/models"
	"fmt"
	"net/http"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func isAdmin(adminIDs []primitive.ObjectID, userID primitive.ObjectID) bool {
	for _, id := range adminIDs {
		if id == userID {
			return true
		}
	}
	return false
}

func (s *Server) CreateGroup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name    string             `json:"name"`
		AdminID primitive.ObjectID `json:"admin_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check if group name already exists
	count, err := s.DB.CountGroupsByName(context.Background(), req.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if count > 0 {
		http.Error(w, "Group name already exists", http.StatusConflict)
		return
	}

	group := models.Group{
		ID:       primitive.NewObjectID(),
		Name:     req.Name,
		AdminIDs: []primitive.ObjectID{req.AdminID},
		JoinCode: generateJoinCode(),
	}

	err = s.DB.CreateGroup(context.Background(), &group)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update admin's family to join this group
	err = s.DB.UpdateFamilyMember(
		context.Background(),
		req.AdminID,
		bson.M{"$push": bson.M{"group_ids": group.ID}},
	)
	if err != nil {
		// Log error but don't fail the request as group is created
		// Ideally we should use a transaction here
		http.Error(w, "Group created but failed to update admin membership", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(group)
}

func (s *Server) JoinGroup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FamilyMemberID primitive.ObjectID `json:"family_id"`
		GroupID        primitive.ObjectID `json:"group_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := s.DB.UpdateFamilyMember(
		context.Background(),
		req.FamilyMemberID,
		bson.M{"$addToSet": bson.M{"group_ids": req.GroupID}},
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) LeaveGroup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FamilyMemberID primitive.ObjectID `json:"family_id"`
		GroupID        primitive.ObjectID `json:"group_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Prevent admin from leaving (they must delete the group)
	group, err := s.DB.GetGroup(context.Background(), req.GroupID)
	if err != nil {
		http.Error(w, "Group not found", http.StatusNotFound)
		return
	}

	if isAdmin(group.AdminIDs, req.FamilyMemberID) {
		http.Error(w, "Admin cannot leave the group. Delete the group instead or remove admin status first.", http.StatusForbidden)
		return
	}

	err = s.DB.UpdateFamilyMember(
		context.Background(),
		req.FamilyMemberID,
		bson.M{"$pull": bson.M{"group_ids": req.GroupID}},
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) JoinGroupByCode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FamilyMemberID primitive.ObjectID `json:"family_id"`
		JoinCode       string             `json:"join_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Find group by join code
	group, err := s.DB.GetGroupByCode(context.Background(), req.JoinCode)
	if err != nil {
		http.Error(w, "Invalid join code", http.StatusNotFound)
		return
	}

	err = s.DB.UpdateFamilyMember(
		context.Background(),
		req.FamilyMemberID,
		bson.M{"$addToSet": bson.M{"group_ids": group.ID}},
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(group)
}

func (s *Server) GetGroupByCode(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	if code == "" {
		http.Error(w, "Missing join code", http.StatusBadRequest)
		return
	}

	group, err := s.DB.GetGroupByCode(context.Background(), code)
	if err != nil {
		http.Error(w, "Group not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(group)
}

func (s *Server) GetGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := s.DB.GetGroups(context.Background())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(groups)
}

func (s *Server) GetGroup(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, "Invalid group id", http.StatusBadRequest)
		return
	}

	group, err := s.DB.GetGroup(context.Background(), id)
	if err != nil {
		http.Error(w, "Group not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(group)
}

func (s *Server) UpdateGroup(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, "Invalid group id", http.StatusBadRequest)
		return
	}

	var req struct {
		Name     string               `json:"name"`
		AdminIDs []primitive.ObjectID `json:"admin_ids"`
		UserID   primitive.ObjectID   `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Verify group exists and user is admin
	group, err := s.DB.GetGroup(context.Background(), id)
	if err != nil {
		http.Error(w, "Group not found", http.StatusNotFound)
		return
	}

	if !isAdmin(group.AdminIDs, req.UserID) {
		http.Error(w, "Unauthorized: Only admin can update group", http.StatusForbidden)
		return
	}

	update := bson.M{}
	if req.Name != "" {
		update["name"] = req.Name
	}
	if len(req.AdminIDs) > 0 {
		update["admin_ids"] = req.AdminIDs
	}

	if len(update) == 0 {
		w.WriteHeader(http.StatusOK)
		return
	}

	err = s.DB.UpdateGroup(context.Background(), id, bson.M{"$set": update})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) DeleteGroup(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, "Invalid group id", http.StatusBadRequest)
		return
	}

	adminIDStr := r.URL.Query().Get("admin_id")
	if adminIDStr == "" {
		http.Error(w, "Missing admin_id", http.StatusBadRequest)
		return
	}
	adminID, err := primitive.ObjectIDFromHex(adminIDStr)
	if err != nil {
		http.Error(w, "Invalid admin_id", http.StatusBadRequest)
		return
	}

	// Verify group exists and user is admin
	group, err := s.DB.GetGroup(context.Background(), id)
	if err != nil {
		http.Error(w, "Group not found", http.StatusNotFound)
		return
	}

	if !isAdmin(group.AdminIDs, adminID) {
		http.Error(w, "Unauthorized: Only admin can delete group", http.StatusForbidden)
		return
	}

	// Delete group
	err = s.DB.DeleteGroup(context.Background(), id)
	if err != nil {
		http.Error(w, "Failed to delete group", http.StatusInternalServerError)
		return
	}

	// Remove group_id from all families
	err = s.DB.RemoveGroupIDFromAllFamilyMembers(context.Background(), id)
	if err != nil {
		// Log error but success for group deletion
		fmt.Printf("Failed to remove group from families: %v\n", err)
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) GetGroupMembers(w http.ResponseWriter, r *http.Request) {
	groupIDStr := r.URL.Query().Get("group_id")
	if groupIDStr == "" {
		http.Error(w, "group_id is required", http.StatusBadRequest)
		return
	}

	groupID, err := primitive.ObjectIDFromHex(groupIDStr)
	if err != nil {
		http.Error(w, "invalid group_id", http.StatusBadRequest)
		return
	}

	familyMembers, err := s.DB.GetFamilyMembersByGroupID(context.Background(), groupID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch households for these families
	householdIDs := make([]primitive.ObjectID, 0)
	seen := make(map[string]bool)
	for _, f := range familyMembers {
		if f.HouseholdID != nil {
			if !seen[f.HouseholdID.Hex()] {
				householdIDs = append(householdIDs, *f.HouseholdID)
				seen[f.HouseholdID.Hex()] = true
			}
		}
	}

	var households []models.Household
	if len(householdIDs) > 0 {
		// We need a GetHouseholdsByIDs method in DB or just loop (inefficient but fine for now)
		// For now, let's just loop as I didn't add GetHouseholdsByIDs
		for _, hID := range householdIDs {
			h, err := s.DB.GetHousehold(context.Background(), hID)
			if err == nil {
				households = append(households, *h)
			}
		}
	}

	safeFamilies := make([]models.SafeFamilyMember, len(familyMembers))
	for i, f := range familyMembers {
		safeFamilies[i] = f.ToSafe()
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"families":   safeFamilies,
		"households": households,
	})
}
