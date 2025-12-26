package handlers

import (
	"context"
	"encoding/json"
	"family-potluck/backend/internal/models"
	"net/http"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (s *Server) AddDish(w http.ResponseWriter, r *http.Request) {
	var dish models.Dish
	if err := json.NewDecoder(r.Body).Decode(&dish); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dish.ID = primitive.NewObjectID()
	err := s.DB.CreateDish(context.Background(), &dish)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Broadcast update
	msg := map[string]interface{}{
		"type": "dish_added",
		"data": dish,
	}
	msgBytes, _ := json.Marshal(msg)
	s.Hub.Broadcast(msgBytes)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(dish)
}

func (s *Server) GetDishes(w http.ResponseWriter, r *http.Request) {
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

	dishes, err := s.DB.GetDishesByEventID(context.Background(), eventID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Collect bringer IDs
	bringerIDs := []primitive.ObjectID{}
	for _, dish := range dishes {
		if dish.BringerID != nil {
			bringerIDs = append(bringerIDs, *dish.BringerID)
		}
	}

	// Fetch families if there are any bringers
	bringerNames := make(map[primitive.ObjectID]string)
	if len(bringerIDs) > 0 {
		familyMembers, err := s.DB.GetFamilyMembersByIDs(context.Background(), bringerIDs)
		if err == nil {
			for _, familyMember := range familyMembers {
				bringerNames[familyMember.ID] = familyMember.Name
			}
		}
	}

	// Populate BringerName
	for i := range dishes {
		if dishes[i].BringerID != nil {
			if name, ok := bringerNames[*dishes[i].BringerID]; ok {
				dishes[i].BringerName = name
			}
		}
	}

	json.NewEncoder(w).Encode(dishes)
}

func (s *Server) PledgeDish(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, "Invalid dish id", http.StatusBadRequest)
		return
	}

	var req struct {
		FamilyMemberID primitive.ObjectID `json:"family_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Fetch dish to get event_id
	dish, err := s.DB.GetDishByID(context.Background(), id)
	if err != nil {
		http.Error(w, "Dish not found", http.StatusNotFound)
		return
	}

	err = s.DB.UpdateDish(
		context.Background(),
		id,
		bson.M{"$set": bson.M{"bringer_id": req.FamilyMemberID}},
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch Family Name for broadcast
	familyMember, err := s.DB.GetFamilyMemberByID(context.Background(), req.FamilyMemberID)
	familyName := "Someone"
	if err == nil {
		familyName = familyMember.Name
	}

	// Broadcast update
	msg := map[string]interface{}{
		"type": "dish_pledged",
		"data": map[string]interface{}{
			"dish_id":      id,
			"event_id":     dish.EventID,
			"bringer_id":   req.FamilyMemberID,
			"bringer_name": familyName,
			"dish_name":    dish.Name,
		},
	}
	msgBytes, _ := json.Marshal(msg)
	s.Hub.Broadcast(msgBytes)

	w.WriteHeader(http.StatusOK)
}

func (s *Server) UnpledgeDish(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, "Invalid dish id", http.StatusBadRequest)
		return
	}

	// Fetch dish to get event_id
	dish, err := s.DB.GetDishByID(context.Background(), id)
	if err != nil {
		http.Error(w, "Dish not found", http.StatusNotFound)
		return
	}

	err = s.DB.UpdateDish(
		context.Background(),
		id,
		bson.M{"$unset": bson.M{"bringer_id": ""}},
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Broadcast update
	msg := map[string]interface{}{
		"type": "dish_unpledged",
		"data": map[string]interface{}{
			"dish_id":   id,
			"event_id":  dish.EventID,
			"dish_name": dish.Name,
		},
	}
	msgBytes, _ := json.Marshal(msg)
	s.Hub.Broadcast(msgBytes)

	// If it was a suggested dish, we might want to delete it if unpledged?
	// No, let's keep it as a suggestion again.
	// But we need to make sure IsSuggested is still true if it was originally suggested.

	w.WriteHeader(http.StatusOK)
}

func (s *Server) DeleteDish(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, "Invalid dish id", http.StatusBadRequest)
		return
	}

	// Fetch dish to get event_id
	dish, err := s.DB.GetDishByID(context.Background(), id)
	if err != nil {
		http.Error(w, "Dish not found", http.StatusNotFound)
		return
	}

	// Permission check
	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		http.Error(w, "Missing user_id", http.StatusBadRequest)
		return
	}
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user_id", http.StatusBadRequest)
		return
	}

	event, err := s.DB.GetEvent(context.Background(), dish.EventID)
	if err != nil {
		http.Error(w, "Event not found", http.StatusInternalServerError)
		return
	}

	group, err := s.DB.GetGroup(context.Background(), event.GroupID)
	if err != nil {
		http.Error(w, "Group not found", http.StatusInternalServerError)
		return
	}

	isAuthorized := false
	if group.AdminID == userID || event.HostID == userID || (dish.BringerID != nil && *dish.BringerID == userID) {
		isAuthorized = true
	} else {
		// Check if userID is in host's household
		userMember, errUser := s.DB.GetFamilyMemberByID(context.Background(), userID)
		hostMember, errHost := s.DB.GetFamilyMemberByID(context.Background(), event.HostID)
		if errUser == nil && errHost == nil && userMember.HouseholdID != nil && hostMember.HouseholdID != nil && *userMember.HouseholdID == *hostMember.HouseholdID {
			isAuthorized = true
		}
	}

	if !isAuthorized {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	err = s.DB.DeleteDish(context.Background(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Broadcast update
	msg := map[string]interface{}{
		"type": "dish_deleted",
		"data": map[string]interface{}{
			"dish_id":  id,
			"event_id": dish.EventID,
		},
	}
	msgBytes, _ := json.Marshal(msg)
	s.Hub.Broadcast(msgBytes)

	w.WriteHeader(http.StatusOK)
}
