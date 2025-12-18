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

func TestAddDish(t *testing.T) {
	mockDB := &database.MockService{}
	hub := websocket.NewHub()
	go hub.Run()
	server := NewServer(mockDB, hub)

	eventID := primitive.NewObjectID()
	dishName := "Test Dish"

	mockDB.CreateDishFunc = func(ctx context.Context, dish *models.Dish) error {
		return nil
	}

	dishReq := models.Dish{
		EventID: eventID,
		Name:    dishName,
	}
	body, _ := json.Marshal(dishReq)

	req, _ := http.NewRequest("POST", "/dishes", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	server.AddDish(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	var resp models.Dish
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Name != dishName {
		t.Errorf("expected dish name %v, got %v", dishName, resp.Name)
	}
}

func TestGetDishes(t *testing.T) {
	mockDB := &database.MockService{}
	server := NewServer(mockDB, nil)

	eventID := primitive.NewObjectID()
	dishes := []models.Dish{
		{ID: primitive.NewObjectID(), Name: "Dish 1", EventID: eventID},
		{ID: primitive.NewObjectID(), Name: "Dish 2", EventID: eventID},
	}

	mockDB.GetDishesByEventIDFunc = func(ctx context.Context, id primitive.ObjectID) ([]models.Dish, error) {
		return dishes, nil
	}

	req, _ := http.NewRequest("GET", "/dishes?event_id="+eventID.Hex(), nil)
	rr := httptest.NewRecorder()

	server.GetDishes(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var resp []models.Dish
	json.NewDecoder(rr.Body).Decode(&resp)
	if len(resp) != 2 {
		t.Errorf("expected 2 dishes, got %v", len(resp))
	}
}
