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

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestCreateSwapRequest(t *testing.T) {
	mockDB := &database.MockService{}
	hub := websocket.NewHub()
	go hub.Run()
	server := NewServer(mockDB, hub)

	eventID := primitive.NewObjectID()
	familyID := primitive.NewObjectID()

	mockDB.CreateSwapRequestFunc = func(ctx context.Context, swap *models.SwapRequest) error {
		return nil
	}

	swapReq := models.SwapRequest{
		EventID:            eventID,
		RequestingFamilyID: familyID,
		Type:               "host",
	}
	body, _ := json.Marshal(swapReq)

	req, _ := http.NewRequest("POST", "/swaps", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	server.CreateSwapRequest(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	var resp models.SwapRequest
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Type != "host" {
		t.Errorf("expected type host, got %v", resp.Type)
	}
}

func TestGetSwapRequests(t *testing.T) {
	mockDB := &database.MockService{}
	server := NewServer(mockDB, nil)

	eventID := primitive.NewObjectID()
	requests := []models.SwapRequest{
		{ID: primitive.NewObjectID(), EventID: eventID, RequestingFamilyID: primitive.NewObjectID()},
		{ID: primitive.NewObjectID(), EventID: eventID, RequestingFamilyID: primitive.NewObjectID()},
	}

	mockDB.GetSwapRequestsByEventIDFunc = func(ctx context.Context, id primitive.ObjectID) ([]models.SwapRequest, error) {
		return requests, nil
	}

	mockDB.GetFamilyByIDFunc = func(ctx context.Context, id primitive.ObjectID) (*models.Family, error) {
		return &models.Family{ID: id, Name: "Test Family"}, nil
	}

	req, _ := http.NewRequest("GET", "/swaps?event_id="+eventID.Hex(), nil)
	rr := httptest.NewRecorder()

	server.GetSwapRequests(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var resp []models.SwapRequest
	json.NewDecoder(rr.Body).Decode(&resp)
	if len(resp) != 2 {
		t.Errorf("expected 2 requests, got %v", len(resp))
	}
}
