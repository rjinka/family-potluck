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

func TestCreateGroup(t *testing.T) {
	mockDB := &database.MockService{}
	server := NewServer(mockDB, nil)

	adminID := primitive.NewObjectID()
	groupName := "Test Group"

	mockDB.CountGroupsByNameFunc = func(ctx context.Context, name string) (int64, error) {
		return 0, nil
	}

	mockDB.CreateGroupFunc = func(ctx context.Context, group *models.Group) error {
		return nil
	}

	mockDB.UpdateFamilyFunc = func(ctx context.Context, id primitive.ObjectID, update bson.M) error {
		return nil
	}

	groupReq := models.Group{
		Name:    groupName,
		AdminID: adminID,
	}
	body, _ := json.Marshal(groupReq)

	req, _ := http.NewRequest("POST", "/groups", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	server.CreateGroup(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	var resp models.Group
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Name != groupName {
		t.Errorf("expected group name %v, got %v", groupName, resp.Name)
	}
	if resp.JoinCode == "" {
		t.Error("expected join code to be generated")
	}
}

func TestGetGroups(t *testing.T) {
	mockDB := &database.MockService{}
	server := NewServer(mockDB, nil)

	groups := []models.Group{
		{ID: primitive.NewObjectID(), Name: "Group 1"},
		{ID: primitive.NewObjectID(), Name: "Group 2"},
	}

	mockDB.GetGroupsFunc = func(ctx context.Context) ([]models.Group, error) {
		return groups, nil
	}

	req, _ := http.NewRequest("GET", "/groups", nil)
	rr := httptest.NewRecorder()

	server.GetGroups(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var resp []models.Group
	json.NewDecoder(rr.Body).Decode(&resp)
	if len(resp) != 2 {
		t.Errorf("expected 2 groups, got %v", len(resp))
	}
}

func TestDeleteGroup_Unauthorized(t *testing.T) {
	mockDB := &database.MockService{}
	server := NewServer(mockDB, nil)

	groupID := primitive.NewObjectID()
	adminID := primitive.NewObjectID()
	otherUserID := primitive.NewObjectID()

	mockDB.GetGroupFunc = func(ctx context.Context, id primitive.ObjectID) (*models.Group, error) {
		return &models.Group{ID: groupID, AdminID: adminID}, nil
	}

	req, _ := http.NewRequest("DELETE", "/groups/"+groupID.Hex()+"?admin_id="+otherUserID.Hex(), nil)
	req.SetPathValue("id", groupID.Hex())
	rr := httptest.NewRecorder()

	server.DeleteGroup(rr, req)

	if status := rr.Code; status != http.StatusForbidden {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusForbidden)
	}
}
