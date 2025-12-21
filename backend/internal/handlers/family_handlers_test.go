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

func TestGetFamilyMember(t *testing.T) {
	mockDB := &database.MockService{}
	server := NewServer(mockDB, nil)

	familyID := primitive.NewObjectID()
	familyMember := &models.FamilyMember{
		ID:    familyID,
		Name:  "Test Family",
		Email: "test@example.com",
	}

	mockDB.GetFamilyMemberByIDFunc = func(ctx context.Context, id primitive.ObjectID) (*models.FamilyMember, error) {
		return familyMember, nil
	}

	req, _ := http.NewRequest("GET", "/families?id="+familyID.Hex(), nil)
	rr := httptest.NewRecorder()

	server.GetFamilyMember(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var resp models.FamilyMember
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.ID != familyID {
		t.Errorf("expected family ID %v, got %v", familyID, resp.ID)
	}
}

func TestUpdateFamilyMember(t *testing.T) {
	mockDB := &database.MockService{}
	server := NewServer(mockDB, nil)

	familyID := primitive.NewObjectID()
	updateData := map[string]interface{}{
		"dietary_preferences": "Vegetarian",
	}
	body, _ := json.Marshal(updateData)

	mockDB.UpdateFamilyMemberFunc = func(ctx context.Context, id primitive.ObjectID, update bson.M) error {
		return nil
	}

	req, _ := http.NewRequest("PATCH", "/families/"+familyID.Hex(), bytes.NewBuffer(body))
	req.SetPathValue("id", familyID.Hex())
	rr := httptest.NewRecorder()

	server.UpdateFamilyMember(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}
