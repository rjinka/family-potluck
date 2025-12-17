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
	collection := s.DB.GetCollection("dishes")
	_, err := collection.InsertOne(context.Background(), dish)
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

	collection := s.DB.GetCollection("dishes")
	cursor, err := collection.Find(context.Background(), bson.M{"event_id": eventID})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.Background())

	var dishes []models.Dish
	if err = cursor.All(context.Background(), &dishes); err != nil {
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
		familiesCollection := s.DB.GetCollection("families")
		filter := bson.M{"_id": bson.M{"$in": bringerIDs}}
		cursor, err := familiesCollection.Find(context.Background(), filter)
		if err == nil {
			defer cursor.Close(context.Background())
			var families []models.Family
			if err = cursor.All(context.Background(), &families); err == nil {
				for _, family := range families {
					bringerNames[family.ID] = family.Name
				}
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
		FamilyID primitive.ObjectID `json:"family_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	collection := s.DB.GetCollection("dishes")

	// Fetch dish to get event_id
	var dish models.Dish
	err = collection.FindOne(context.Background(), bson.M{"_id": id}).Decode(&dish)
	if err != nil {
		http.Error(w, "Dish not found", http.StatusNotFound)
		return
	}

	_, err = collection.UpdateOne(
		context.Background(),
		bson.M{"_id": id},
		bson.M{"$set": bson.M{"bringer_id": req.FamilyID}},
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch Family Name for broadcast
	familiesCollection := s.DB.GetCollection("families")
	var family models.Family
	err = familiesCollection.FindOne(context.Background(), bson.M{"_id": req.FamilyID}).Decode(&family)
	familyName := "Someone"
	if err == nil {
		familyName = family.Name
	}

	// Broadcast update
	msg := map[string]interface{}{
		"type": "dish_pledged",
		"data": map[string]interface{}{
			"dish_id":      id,
			"event_id":     dish.EventID,
			"bringer_id":   req.FamilyID,
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

	collection := s.DB.GetCollection("dishes")

	// Fetch dish to get event_id
	var dish models.Dish
	err = collection.FindOne(context.Background(), bson.M{"_id": id}).Decode(&dish)
	if err != nil {
		http.Error(w, "Dish not found", http.StatusNotFound)
		return
	}

	_, err = collection.UpdateOne(
		context.Background(),
		bson.M{"_id": id},
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

	w.WriteHeader(http.StatusOK)
}

func (s *Server) DeleteDish(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, "Invalid dish id", http.StatusBadRequest)
		return
	}

	collection := s.DB.GetCollection("dishes")

	// Fetch dish to get event_id
	var dish models.Dish
	err = collection.FindOne(context.Background(), bson.M{"_id": id}).Decode(&dish)
	if err != nil {
		http.Error(w, "Dish not found", http.StatusNotFound)
		return
	}

	_, err = collection.DeleteOne(context.Background(), bson.M{"_id": id})
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
