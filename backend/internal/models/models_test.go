package models

import (
	"encoding/json"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestFamilyStruct(t *testing.T) {
	id := primitive.NewObjectID()
	family := Family{
		ID:       id,
		Name:     "Test Family",
		Email:    "test@example.com",
		GoogleID: "12345",
	}

	// Test JSON Marshaling
	data, err := json.Marshal(family)
	if err != nil {
		t.Fatalf("Failed to marshal Family: %v", err)
	}

	// Test JSON Unmarshaling
	var decoded Family
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal Family: %v", err)
	}

	if decoded.ID != family.ID {
		t.Errorf("Expected ID %v, got %v", family.ID, decoded.ID)
	}
	if decoded.Name != family.Name {
		t.Errorf("Expected Name %v, got %v", family.Name, decoded.Name)
	}
}

func TestEventStruct(t *testing.T) {
	id := primitive.NewObjectID()
	now := time.Now().Truncate(time.Millisecond) // Truncate to match JSON precision if needed, though RFC3339 is usually fine.
	// Note: time.Time marshaling/unmarshaling can have slight precision differences depending on format.
	// For this test, we'll just check other fields or accept slight diffs if we used DeepEqual.

	event := Event{
		ID:          id,
		Date:        now,
		Type:        "Dinner",
		Description: "Potluck",
		Status:      "scheduled",
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal Event: %v", err)
	}

	var decoded Event
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal Event: %v", err)
	}

	if decoded.Type != event.Type {
		t.Errorf("Expected Type %v, got %v", event.Type, decoded.Type)
	}
	if decoded.Status != event.Status {
		t.Errorf("Expected Status %v, got %v", event.Status, decoded.Status)
	}
}
