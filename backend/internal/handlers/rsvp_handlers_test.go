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

func TestRSVPEvent(t *testing.T) {
	mockDB := &database.MockService{}
	hub := websocket.NewHub()
	go hub.Run()
	server := NewServer(mockDB, hub)

	eventID := primitive.NewObjectID()
	familyID := primitive.NewObjectID()
	rsvpID := primitive.NewObjectID()

	mockDB.UpsertRSVPFunc = func(ctx context.Context, rsvp *models.RSVP) (primitive.ObjectID, error) {
		return rsvpID, nil
	}

	mockDB.GetFamilyByIDFunc = func(ctx context.Context, id primitive.ObjectID) (*models.Family, error) {
		return &models.Family{ID: familyID, Name: "Test Family"}, nil
	}

	rsvpReq := models.RSVP{
		EventID:  eventID,
		FamilyID: familyID,
		Status:   "Yes",
		Count:    2,
	}
	body, _ := json.Marshal(rsvpReq)

	req, _ := http.NewRequest("POST", "/rsvps", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	server.RSVPEvent(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var resp models.RSVP
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.ID != rsvpID {
		t.Errorf("expected rsvp ID %v, got %v", rsvpID, resp.ID)
	}
	if resp.FamilyName != "Test Family" {
		t.Errorf("expected family name Test Family, got %v", resp.FamilyName)
	}
}

func TestGetRSVPs(t *testing.T) {
	mockDB := &database.MockService{}
	server := NewServer(mockDB, nil)

	eventID := primitive.NewObjectID()
	rsvps := []models.RSVP{
		{ID: primitive.NewObjectID(), EventID: eventID, FamilyID: primitive.NewObjectID()},
		{ID: primitive.NewObjectID(), EventID: eventID, FamilyID: primitive.NewObjectID()},
	}

	mockDB.GetRSVPsByEventIDFunc = func(ctx context.Context, id primitive.ObjectID) ([]models.RSVP, error) {
		return rsvps, nil
	}

	mockDB.GetFamilyByIDFunc = func(ctx context.Context, id primitive.ObjectID) (*models.Family, error) {
		return &models.Family{ID: id, Name: "Test Family"}, nil
	}

	req, _ := http.NewRequest("GET", "/rsvps?event_id="+eventID.Hex(), nil)
	rr := httptest.NewRecorder()

	server.GetRSVPs(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var resp []models.RSVP
	json.NewDecoder(rr.Body).Decode(&resp)
	if len(resp) != 2 {
		t.Errorf("expected 2 rsvps, got %v", len(resp))
	}
}
