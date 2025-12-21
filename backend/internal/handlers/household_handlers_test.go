package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"family-potluck/backend/internal/database"
	"family-potluck/backend/internal/models"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestCreateHousehold(t *testing.T) {
	mockDB := &database.MockService{}
	server := NewServer(mockDB, nil)

	mockDB.CreateHouseholdFunc = func(ctx context.Context, household *models.Household) error {
		return nil
	}
	mockDB.UpdateFamilyMemberFunc = func(ctx context.Context, id primitive.ObjectID, update bson.M) error {
		return nil
	}

	payload := map[string]interface{}{
		"name":      "The Smiths",
		"family_id": primitive.NewObjectID(),
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/households", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	server.CreateHousehold(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status Created, got %v", w.Code)
	}
}

func TestJoinHousehold(t *testing.T) {
	mockDB := &database.MockService{}
	server := NewServer(mockDB, nil)

	mockDB.UpdateHouseholdFunc = func(ctx context.Context, id primitive.ObjectID, update bson.M) error {
		return nil
	}
	mockDB.UpdateFamilyMemberFunc = func(ctx context.Context, id primitive.ObjectID, update bson.M) error {
		return nil
	}

	payload := map[string]interface{}{
		"household_id": primitive.NewObjectID(),
		"family_id":    primitive.NewObjectID(),
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/households/join", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	server.JoinHousehold(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %v", w.Code)
	}
}

func TestGetHousehold(t *testing.T) {
	mockDB := &database.MockService{}
	server := NewServer(mockDB, nil)

	householdID := primitive.NewObjectID()
	mockDB.GetHouseholdFunc = func(ctx context.Context, id primitive.ObjectID) (*models.Household, error) {
		return &models.Household{ID: householdID, Name: "The Smiths"}, nil
	}

	req := httptest.NewRequest("GET", "/households/"+householdID.Hex(), nil)
	req.SetPathValue("id", householdID.Hex())
	w := httptest.NewRecorder()

	server.GetHousehold(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %v", w.Code)
	}
}

func TestDeleteHousehold(t *testing.T) {
	mockDB := &database.MockService{}
	server := NewServer(mockDB, nil)

	householdID := primitive.NewObjectID()
	mockDB.DeleteHouseholdFunc = func(ctx context.Context, id primitive.ObjectID) error {
		return nil
	}

	req := httptest.NewRequest("DELETE", "/households/"+householdID.Hex(), nil)
	req.SetPathValue("id", householdID.Hex())
	w := httptest.NewRecorder()

	server.DeleteHousehold(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %v", w.Code)
	}
}

func TestRemoveMemberFromHousehold(t *testing.T) {
	mockDB := &database.MockService{}
	server := NewServer(mockDB, nil)

	mockDB.RemoveMemberFromHouseholdFunc = func(ctx context.Context, householdID, familyID primitive.ObjectID) error {
		return nil
	}

	payload := map[string]interface{}{
		"household_id": primitive.NewObjectID(),
		"family_id":    primitive.NewObjectID(),
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/households/remove-member", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	server.RemoveMemberFromHousehold(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %v", w.Code)
	}
}
