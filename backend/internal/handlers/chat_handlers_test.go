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

func TestSendChatMessage(t *testing.T) {
	mockDB := &database.MockService{}
	hub := websocket.NewHub()
	go hub.Run()
	server := NewServer(mockDB, hub)

	eventID := primitive.NewObjectID()
	familyID := primitive.NewObjectID()
	groupID := primitive.NewObjectID()

	mockDB.GetEventFunc = func(ctx context.Context, id primitive.ObjectID) (*models.Event, error) {
		return &models.Event{ID: eventID, GroupID: groupID}, nil
	}

	mockDB.GetFamilyMemberByIDFunc = func(ctx context.Context, id primitive.ObjectID) (*models.FamilyMember, error) {
		return &models.FamilyMember{ID: familyID, Name: "Test Family", GroupIDs: []primitive.ObjectID{groupID}}, nil
	}

	mockDB.CreateChatMessageFunc = func(ctx context.Context, msg *models.ChatMessage) error {
		return nil
	}

	msgReq := models.ChatMessage{
		EventID:        eventID,
		FamilyMemberID: familyID,
		Content:        "Hello",
	}
	body, _ := json.Marshal(msgReq)

	req, _ := http.NewRequest("POST", "/chat", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	server.SendChatMessage(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	var resp models.ChatMessage
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Content != "Hello" {
		t.Errorf("expected content Hello, got %v", resp.Content)
	}
}

func TestGetChatMessages(t *testing.T) {
	mockDB := &database.MockService{}
	server := NewServer(mockDB, nil)

	eventID := primitive.NewObjectID()
	messages := []models.ChatMessage{
		{ID: primitive.NewObjectID(), EventID: eventID, Content: "Msg 1"},
		{ID: primitive.NewObjectID(), EventID: eventID, Content: "Msg 2"},
	}

	mockDB.GetChatMessagesByEventIDFunc = func(ctx context.Context, id primitive.ObjectID) ([]models.ChatMessage, error) {
		return messages, nil
	}

	req, _ := http.NewRequest("GET", "/chat?event_id="+eventID.Hex(), nil)
	rr := httptest.NewRecorder()

	server.GetChatMessages(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var resp []models.ChatMessage
	json.NewDecoder(rr.Body).Decode(&resp)
	if len(resp) != 2 {
		t.Errorf("expected 2 messages, got %v", len(resp))
	}
}
