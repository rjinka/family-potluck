package handlers

import (
	"context"
	"encoding/json"
	"family-potluck/backend/internal/models"
	"fmt"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (s *Server) CreateSwapRequest(w http.ResponseWriter, r *http.Request) {
	var req models.SwapRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	req.ID = primitive.NewObjectID()
	req.Status = "pending"
	req.CreatedAt = time.Now()

	collection := s.DB.GetCollection("swap_requests")
	_, err := collection.InsertOne(context.Background(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Broadcast update
	msg := map[string]interface{}{
		"type": "swap_created",
		"data": req,
	}
	msgBytes, _ := json.Marshal(msg)
	s.Hub.Broadcast(msgBytes)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(req)
}

func (s *Server) UpdateSwapRequest(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var update struct {
		Status         string              `json:"status"`
		TargetFamilyID *primitive.ObjectID `json:"target_family_id"`
		EventUpdates   *struct {
			Date     time.Time `json:"date"`
			Location string    `json:"location"`
		} `json:"event_updates"`
	}
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	collection := s.DB.GetCollection("swap_requests")

	// Fetch request to get details for broadcast
	var req models.SwapRequest
	err = collection.FindOne(context.Background(), bson.M{"_id": id}).Decode(&req)
	if err != nil {
		http.Error(w, "Swap request not found", http.StatusNotFound)
		return
	}

	updateFields := bson.M{"status": update.Status}
	if update.TargetFamilyID != nil {
		updateFields["target_family_id"] = update.TargetFamilyID
		req.TargetFamilyID = update.TargetFamilyID // Update local object for broadcast
	}

	_, err = collection.UpdateOne(
		context.Background(),
		bson.M{"_id": id},
		bson.M{"$set": updateFields},
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update the request object with new status for broadcast
	req.Status = update.Status

	// If approved, update the dish bringer or event host
	if req.Status == "approved" {
		if req.Type == "host" {
			// Update Event Host
			eventsCollection := s.DB.GetCollection("events")

			// Let's check if we have a TargetID.
			var newHostID primitive.ObjectID
			if req.TargetFamilyID != nil {
				newHostID = *req.TargetFamilyID
			}

			eventUpdateDoc := bson.M{"host_id": newHostID}
			if update.EventUpdates != nil {
				if !update.EventUpdates.Date.IsZero() {
					eventUpdateDoc["date"] = update.EventUpdates.Date
				}
				if update.EventUpdates.Location != "" {
					eventUpdateDoc["location"] = update.EventUpdates.Location
				}
			}

			_, err = eventsCollection.UpdateOne(
				context.Background(),
				bson.M{"_id": req.EventID},
				bson.M{"$set": eventUpdateDoc},
			)
			if err != nil {
				fmt.Printf("Failed to update event host: %v\n", err)
			} else {
				// Broadcast event update
				var updatedEvent models.Event
				eventsCollection.FindOne(context.Background(), bson.M{"_id": req.EventID}).Decode(&updatedEvent)

				eventMsg := map[string]interface{}{
					"type": "event_updated",
					"data": updatedEvent,
				}
				eventMsgBytes, _ := json.Marshal(eventMsg)
				s.Hub.Broadcast(eventMsgBytes)
			}

		} else {
			// Default to dish swap
			dishesCollection := s.DB.GetCollection("dishes")

			// Fetch dish to check current bringer to determine direction (Request vs Offer)
			var currentDish models.Dish
			err = dishesCollection.FindOne(context.Background(), bson.M{"_id": req.DishID}).Decode(&currentDish)

			var newBringerID primitive.ObjectID
			if err == nil && currentDish.BringerID != nil && *currentDish.BringerID == req.RequestingFamilyID {
				// Requester is the current bringer -> It's an Offer -> Target (Acceptor) becomes bringer
				if req.TargetFamilyID != nil {
					newBringerID = *req.TargetFamilyID
				} else {
					// Fallback or error handling if needed, but TargetFamilyID should be set on acceptance
					newBringerID = req.RequestingFamilyID // Revert to requester if no target? Or fail?
				}
			} else {
				// Requester is NOT the current bringer -> It's a Request -> Requester becomes bringer
				newBringerID = req.RequestingFamilyID
			}

			_, err = dishesCollection.UpdateOne(
				context.Background(),
				bson.M{"_id": req.DishID},
				bson.M{"$set": bson.M{"bringer_id": newBringerID}},
			)
			if err != nil {
				// Log error but continue
				fmt.Printf("Failed to update dish bringer: %v\n", err)
			} else {
				// Broadcast dish update
				dishMsg := map[string]interface{}{
					"type": "dish_pledged",
					"data": map[string]interface{}{
						"dish_id":    req.DishID,
						"event_id":   req.EventID,
						"bringer_id": req.RequestingFamilyID,
					},
				}
				dishMsgBytes, _ := json.Marshal(dishMsg)
				s.Hub.Broadcast(dishMsgBytes)
			}
		}
	}

	// Broadcast update
	msg := map[string]interface{}{
		"type": "swap_updated",
		"data": req,
	}
	msgBytes, _ := json.Marshal(msg)
	s.Hub.Broadcast(msgBytes)

	w.WriteHeader(http.StatusOK)
}

func (s *Server) GetSwapRequests(w http.ResponseWriter, r *http.Request) {
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

	collection := s.DB.GetCollection("swap_requests")
	cursor, err := collection.Find(context.Background(), bson.M{"event_id": eventID})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.Background())

	var requests []models.SwapRequest
	if err = cursor.All(context.Background(), &requests); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Populate family names
	familiesCollection := s.DB.GetCollection("families")
	for i := range requests {
		var requestingFamily models.Family
		err := familiesCollection.FindOne(context.Background(), bson.M{"_id": requests[i].RequestingFamilyID}).Decode(&requestingFamily)
		if err == nil {
			requests[i].RequestingFamilyName = requestingFamily.Name
		}

		if requests[i].TargetFamilyID != nil {
			var targetFamily models.Family
			err := familiesCollection.FindOne(context.Background(), bson.M{"_id": requests[i].TargetFamilyID}).Decode(&targetFamily)
			if err == nil {
				requests[i].TargetFamilyName = targetFamily.Name
			}
		}
	}

	json.NewEncoder(w).Encode(requests)
}
