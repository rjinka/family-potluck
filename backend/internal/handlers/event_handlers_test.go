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

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestCreateEvent(t *testing.T) {
	mockDB := &database.MockService{}
	hub := websocket.NewHub()
	go hub.Run()
	server := NewServer(mockDB, hub)

	groupID := primitive.NewObjectID()
	hostID := primitive.NewObjectID()

	mockDB.CreateEventFunc = func(ctx context.Context, event *models.Event) error {
		return nil
	}
	mockDB.GetFamilyMemberByIDFunc = func(ctx context.Context, id primitive.ObjectID) (*models.FamilyMember, error) {
		return &models.FamilyMember{ID: id}, nil
	}

	eventReq := models.Event{
		GroupID:     groupID,
		HostID:      hostID,
		Description: "Test Event",
		Date:        time.Now().Add(24 * time.Hour),
	}
	body, _ := json.Marshal(eventReq)

	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	server.CreateEvent(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	var resp models.Event
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Description != "Test Event" {
		t.Errorf("expected description Test Event, got %v", resp.Description)
	}
	if resp.GuestJoinCode == "" {
		t.Error("expected guest join code to be generated")
	}
}

func TestGetEvents(t *testing.T) {
	mockDB := &database.MockService{}
	server := NewServer(mockDB, nil)

	groupID := primitive.NewObjectID()
	events := []models.Event{
		{ID: primitive.NewObjectID(), Description: "Event 1", GroupID: groupID},
		{ID: primitive.NewObjectID(), Description: "Event 2", GroupID: groupID},
	}

	mockDB.GetEventsByGroupIDFunc = func(ctx context.Context, id primitive.ObjectID, includeCompleted bool) ([]models.Event, error) {
		return events, nil
	}

	req, _ := http.NewRequest("GET", "/events?group_id="+groupID.Hex(), nil)
	rr := httptest.NewRecorder()

	server.GetEvents(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var resp []models.Event
	json.NewDecoder(rr.Body).Decode(&resp)
	if len(resp) != 2 {
		t.Errorf("expected 2 events, got %v", len(resp))
	}
}

func TestDeleteEvent_Unauthorized(t *testing.T) {
	mockDB := &database.MockService{}
	server := NewServer(mockDB, nil)

	eventID := primitive.NewObjectID()
	groupID := primitive.NewObjectID()
	adminID := primitive.NewObjectID()
	otherUserID := primitive.NewObjectID()

	mockDB.GetEventFunc = func(ctx context.Context, id primitive.ObjectID) (*models.Event, error) {
		return &models.Event{ID: eventID, GroupID: groupID, HostID: adminID}, nil
	}

	mockDB.GetGroupFunc = func(ctx context.Context, id primitive.ObjectID) (*models.Group, error) {
		return &models.Group{ID: groupID, AdminID: adminID}, nil
	}

	req, _ := http.NewRequest("DELETE", "/events/"+eventID.Hex()+"?user_id="+otherUserID.Hex(), nil)
	req.SetPathValue("id", eventID.Hex())
	rr := httptest.NewRecorder()

	server.DeleteEvent(rr, req)

	if status := rr.Code; status != http.StatusForbidden {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusForbidden)
	}
}

func TestCreateEvent_WithHouseholdAddress(t *testing.T) {
	mockDB := &database.MockService{}
	hub := websocket.NewHub()
	go hub.Run()
	server := NewServer(mockDB, hub)

	groupID := primitive.NewObjectID()
	hostID := primitive.NewObjectID()
	householdID := primitive.NewObjectID()
	expectedAddress := "123 Main St"

	mockDB.CreateEventFunc = func(ctx context.Context, event *models.Event) error {
		return nil
	}
	mockDB.GetFamilyMemberByIDFunc = func(ctx context.Context, id primitive.ObjectID) (*models.FamilyMember, error) {
		return &models.FamilyMember{ID: id, HouseholdID: &householdID}, nil
	}
	mockDB.GetHouseholdFunc = func(ctx context.Context, id primitive.ObjectID) (*models.Household, error) {
		return &models.Household{ID: id, Address: expectedAddress}, nil
	}

	eventReq := models.Event{
		GroupID:     groupID,
		HostID:      hostID,
		Description: "Test Event",
		Date:        time.Now().Add(24 * time.Hour),
		Location:    "", // Empty location
	}
	body, _ := json.Marshal(eventReq)

	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	server.CreateEvent(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	var resp models.Event
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Location != expectedAddress {
		t.Errorf("expected location %v, got %v", expectedAddress, resp.Location)
	}
}
