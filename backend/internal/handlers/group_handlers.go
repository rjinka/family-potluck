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

func (s *Server) CreateGroup(w http.ResponseWriter, r *http.Request) {
	var group models.Group
	if err := json.NewDecoder(r.Body).Decode(&group); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check if group name already exists
	collection := s.DB.GetCollection("groups")
	count, err := collection.CountDocuments(context.Background(), bson.M{"name": group.Name})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if count > 0 {
		http.Error(w, "Group name already exists", http.StatusConflict)
		return
	}

	group.ID = primitive.NewObjectID()
	group.JoinCode = generateJoinCode()
	_, err = collection.InsertOne(context.Background(), group)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update admin's family to join this group
	familiesCollection := s.DB.GetCollection("families")
	_, err = familiesCollection.UpdateOne(
		context.Background(),
		bson.M{"_id": group.AdminID},
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
		FamilyID primitive.ObjectID `json:"family_id"`
		GroupID  primitive.ObjectID `json:"group_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	familiesCollection := s.DB.GetCollection("families")
	_, err := familiesCollection.UpdateOne(
		context.Background(),
		bson.M{"_id": req.FamilyID},
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
		FamilyID primitive.ObjectID `json:"family_id"`
		GroupID  primitive.ObjectID `json:"group_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Prevent admin from leaving (they must delete the group)
	groupsCollection := s.DB.GetCollection("groups")
	var group models.Group
	err := groupsCollection.FindOne(context.Background(), bson.M{"_id": req.GroupID}).Decode(&group)
	if err != nil {
		http.Error(w, "Group not found", http.StatusNotFound)
		return
	}

	if group.AdminID == req.FamilyID {
		http.Error(w, "Admin cannot leave the group. Delete the group instead.", http.StatusForbidden)
		return
	}

	familiesCollection := s.DB.GetCollection("families")
	_, err = familiesCollection.UpdateOne(
		context.Background(),
		bson.M{"_id": req.FamilyID},
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
		FamilyID primitive.ObjectID `json:"family_id"`
		JoinCode string             `json:"join_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Find group by join code
	collection := s.DB.GetCollection("groups")
	var group models.Group
	err := collection.FindOne(context.Background(), bson.M{"join_code": req.JoinCode}).Decode(&group)
	if err != nil {
		http.Error(w, "Invalid join code", http.StatusNotFound)
		return
	}

	familiesCollection := s.DB.GetCollection("families")
	_, err = familiesCollection.UpdateOne(
		context.Background(),
		bson.M{"_id": req.FamilyID},
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

	collection := s.DB.GetCollection("groups")
	var group models.Group
	err := collection.FindOne(context.Background(), bson.M{"join_code": code}).Decode(&group)
	if err != nil {
		http.Error(w, "Group not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(group)
}

func (s *Server) GetGroups(w http.ResponseWriter, r *http.Request) {
	collection := s.DB.GetCollection("groups")
	cursor, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.Background())

	var groups []models.Group
	if err = cursor.All(context.Background(), &groups); err != nil {
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

	collection := s.DB.GetCollection("groups")
	var group models.Group
	err = collection.FindOne(context.Background(), bson.M{"_id": id}).Decode(&group)
	if err != nil {
		http.Error(w, "Group not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(group)
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
	groupsCollection := s.DB.GetCollection("groups")
	var group models.Group
	err = groupsCollection.FindOne(context.Background(), bson.M{"_id": id}).Decode(&group)
	if err != nil {
		http.Error(w, "Group not found", http.StatusNotFound)
		return
	}

	if group.AdminID != adminID {
		http.Error(w, "Unauthorized: Only admin can delete group", http.StatusForbidden)
		return
	}

	// Delete group
	_, err = groupsCollection.DeleteOne(context.Background(), bson.M{"_id": id})
	if err != nil {
		http.Error(w, "Failed to delete group", http.StatusInternalServerError)
		return
	}

	// Remove group_id from all families
	familiesCollection := s.DB.GetCollection("families")
	_, err = familiesCollection.UpdateMany(
		context.Background(),
		bson.M{"group_ids": id},
		bson.M{"$pull": bson.M{"group_ids": id}},
	)
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

	collection := s.DB.GetCollection("families")
	cursor, err := collection.Find(context.Background(), bson.M{"group_ids": groupID})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.Background())

	var families []models.Family
	if err = cursor.All(context.Background(), &families); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(families)
}
