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

	err := s.DB.CreateSwapRequest(context.Background(), &req)
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
		Status               string              `json:"status"`
		TargetFamilyMemberID *primitive.ObjectID `json:"target_family_id"`
		EventUpdates         *struct {
			Date     time.Time `json:"date"`
			Location string    `json:"location"`
		} `json:"event_updates"`
	}
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Fetch request to get details for broadcast
	req, err := s.DB.GetSwapRequestByID(context.Background(), id)
	if err != nil {
		http.Error(w, "Swap request not found", http.StatusNotFound)
		return
	}

	updateFields := bson.M{"status": update.Status}
	if update.TargetFamilyMemberID != nil {
		updateFields["target_family_id"] = update.TargetFamilyMemberID
		req.TargetFamilyMemberID = update.TargetFamilyMemberID // Update local object for broadcast
	}

	err = s.DB.UpdateSwapRequest(context.Background(), id, bson.M{"$set": updateFields})
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
			var newHostID primitive.ObjectID
			if req.TargetFamilyMemberID != nil {
				newHostID = *req.TargetFamilyMemberID
			}

			// Fetch current event to check location
			event, err := s.DB.GetEvent(context.Background(), req.EventID)
			if err != nil {
				http.Error(w, "Event not found", http.StatusNotFound)
				return
			}

			eventUpdateDoc := bson.M{"host_id": newHostID}

			// Get old host address for comparison
			oldHost, err := s.DB.GetFamilyMemberByID(context.Background(), event.HostID)
			var oldAddress string
			if err == nil && oldHost.HouseholdID != nil {
				oldHousehold, err := s.DB.GetHousehold(context.Background(), *oldHost.HouseholdID)
				if err == nil {
					oldAddress = oldHousehold.Address
				}
			}

			// Get new host address
			newHost, err := s.DB.GetFamilyMemberByID(context.Background(), newHostID)
			var newAddress string
			if err == nil && newHost.HouseholdID != nil {
				newHousehold, err := s.DB.GetHousehold(context.Background(), *newHost.HouseholdID)
				if err == nil && newHousehold.Address != "" {
					newAddress = newHousehold.Address
				}
			}

			// Initial automatic update based on current state
			if (event.Location == "" || event.Location == oldAddress) && newAddress != "" {
				eventUpdateDoc["location"] = newAddress
			}

			if update.EventUpdates != nil {
				if !update.EventUpdates.Date.IsZero() {
					eventUpdateDoc["date"] = update.EventUpdates.Date
				}
				// If user provided a location that is NOT the old address and NOT empty, use it.
				// If they provided the old address (likely from pre-fill), and we have a new address,
				// we already set it in eventUpdateDoc["location"] above.
				if update.EventUpdates.Location != "" && update.EventUpdates.Location != oldAddress {
					eventUpdateDoc["location"] = update.EventUpdates.Location
				}
			}

			err = s.DB.UpdateEvent(context.Background(), req.EventID, bson.M{"$set": eventUpdateDoc})
			if err != nil {
				fmt.Printf("Failed to update event host: %v\n", err)
			} else {
				// Broadcast event update
				updatedEvent, err := s.DB.GetEvent(context.Background(), req.EventID)
				if err == nil {
					eventMsg := map[string]interface{}{
						"type": "event_updated",
						"data": updatedEvent,
					}
					eventMsgBytes, _ := json.Marshal(eventMsg)
					s.Hub.Broadcast(eventMsgBytes)
				}
			}

		} else {
			// Default to dish swap
			// Fetch dish to check current bringer to determine direction (Request vs Offer)
			currentDish, err := s.DB.GetDishByID(context.Background(), *req.DishID)

			var newBringerID primitive.ObjectID
			if err == nil && currentDish.BringerID != nil && *currentDish.BringerID == req.RequestingFamilyMemberID {
				// Requester is the current bringer -> It's an Offer -> Target (Acceptor) becomes bringer
				if req.TargetFamilyMemberID != nil {
					newBringerID = *req.TargetFamilyMemberID
				} else {
					newBringerID = req.RequestingFamilyMemberID
				}
			} else {
				// Requester is NOT the current bringer -> It's a Request -> Requester becomes bringer
				newBringerID = req.RequestingFamilyMemberID
			}

			err = s.DB.UpdateDish(
				context.Background(),
				*req.DishID,
				bson.M{"$set": bson.M{"bringer_id": newBringerID}},
			)
			if err != nil {
				fmt.Printf("Failed to update dish bringer: %v\n", err)
			} else {
				// Broadcast dish update
				dishMsg := map[string]interface{}{
					"type": "dish_pledged",
					"data": map[string]interface{}{
						"dish_id":    req.DishID,
						"event_id":   req.EventID,
						"bringer_id": req.RequestingFamilyMemberID,
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

	requests, err := s.DB.GetSwapRequestsByEventID(context.Background(), eventID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Populate family names
	for i := range requests {
		requestingFamilyMember, err := s.DB.GetFamilyMemberByID(context.Background(), requests[i].RequestingFamilyMemberID)
		if err == nil {
			requests[i].RequestingFamilyName = requestingFamilyMember.Name
		}

		if requests[i].TargetFamilyMemberID != nil {
			targetFamilyMember, err := s.DB.GetFamilyMemberByID(context.Background(), *requests[i].TargetFamilyMemberID)
			if err == nil {
				requests[i].TargetFamilyName = targetFamilyMember.Name
			}
		}
	}

	json.NewEncoder(w).Encode(requests)
}
