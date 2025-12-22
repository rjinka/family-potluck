package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"family-potluck/backend/internal/database"
	"family-potluck/backend/internal/models"
	"family-potluck/backend/internal/websocket"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestUpdateSwapRequest_HostSwap_LocationUpdate(t *testing.T) {
	mockDB := &database.MockService{}
	hub := websocket.NewHub()
	go hub.Run()
	server := NewServer(mockDB, hub)

	swapID := primitive.NewObjectID()
	eventID := primitive.NewObjectID()
	oldHostID := primitive.NewObjectID()
	newHostID := primitive.NewObjectID()
	oldHouseholdID := primitive.NewObjectID()
	newHouseholdID := primitive.NewObjectID()

	// Mock Swap Request
	mockDB.GetSwapRequestByIDFunc = func(ctx context.Context, id primitive.ObjectID) (*models.SwapRequest, error) {
		return &models.SwapRequest{
			ID:                       swapID,
			EventID:                  eventID,
			Type:                     "host",
			RequestingFamilyMemberID: oldHostID,
			Status:                   "pending",
		}, nil
	}

	// Mock UpdateSwapRequest
	mockDB.UpdateSwapRequestFunc = func(ctx context.Context, id primitive.ObjectID, update bson.M) error {
		return nil
	}

	// Mock GetEvent
	mockDB.GetEventFunc = func(ctx context.Context, id primitive.ObjectID) (*models.Event, error) {
		return &models.Event{
			ID:       eventID,
			HostID:   oldHostID,
			Location: "Old Address", // Matches old household address
		}, nil
	}

	// Mock GetFamilyMemberByID for old and new host
	mockDB.GetFamilyMemberByIDFunc = func(ctx context.Context, id primitive.ObjectID) (*models.FamilyMember, error) {
		if id == oldHostID {
			return &models.FamilyMember{ID: oldHostID, HouseholdID: &oldHouseholdID}, nil
		}
		if id == newHostID {
			return &models.FamilyMember{ID: newHostID, HouseholdID: &newHouseholdID}, nil
		}
		return nil, nil
	}

	// Mock GetHousehold for old and new household
	mockDB.GetHouseholdFunc = func(ctx context.Context, id primitive.ObjectID) (*models.Household, error) {
		if id == oldHouseholdID {
			return &models.Household{ID: oldHouseholdID, Address: "Old Address"}, nil
		}
		if id == newHouseholdID {
			return &models.Household{ID: newHouseholdID, Address: "New Address"}, nil
		}
		return nil, nil
	}

	// Mock UpdateEvent - verify location update
	locationUpdated := false
	mockDB.UpdateEventFunc = func(ctx context.Context, id primitive.ObjectID, update bson.M) error {
		t.Logf("UpdateEvent called with: %+v", update)
		var setDoc map[string]interface{}
		if m, ok := update["$set"].(primitive.M); ok {
			setDoc = map[string]interface{}(m)
		} else if m, ok := update["$set"].(bson.M); ok {
			setDoc = map[string]interface{}(m)
		} else if m, ok := update["$set"].(map[string]interface{}); ok {
			setDoc = m
		} else {
			// Fallback: try to iterate if possible or fail
			t.Logf("update['$set'] has unexpected type: %T", update["$set"])
			return nil
		}

		if loc, ok := setDoc["location"]; ok {
			t.Logf("Location in update: %v", loc)
			if loc == "New Address" {
				locationUpdated = true
			}
		}
		return nil
	}

	// Request Body
	updateBody := map[string]interface{}{
		"status":           "approved",
		"target_family_id": newHostID,
		"event_updates": map[string]interface{}{
			"date":     time.Now(),
			"location": "Old Address", // Frontend sends old address (pre-filled)
		},
	}
	body, _ := json.Marshal(updateBody)

	req, _ := http.NewRequest("PATCH", "/swaps/"+swapID.Hex(), bytes.NewBuffer(body))
	req.SetPathValue("id", swapID.Hex())
	rr := httptest.NewRecorder()

	server.UpdateSwapRequest(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	if !locationUpdated {
		t.Error("expected event location to be updated to New Address")
	}
}
