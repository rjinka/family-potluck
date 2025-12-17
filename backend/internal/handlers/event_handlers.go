package handlers

import (
	"context"
	"encoding/json"
	"family-potluck/backend/internal/models"
	"fmt"
	"net/http"
	"sort"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (s *Server) CreateEvent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		models.Event
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	event := req.Event
	event.ID = primitive.NewObjectID()
	event.GuestJoinCode = generateJoinCode()
	if event.Recurrence != "" {
		event.RecurrenceID = primitive.NewObjectID()
	}

	collection := s.DB.GetCollection("events")
	_, err := collection.InsertOne(context.Background(), event)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Broadcast update
	msg := map[string]interface{}{
		"type": "event_created",
		"data": event,
	}
	msgBytes, _ := json.Marshal(msg)
	s.Hub.Broadcast(msgBytes)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(event)
}
func (s *Server) FinishEvent(w http.ResponseWriter, r *http.Request) {
	fmt.Println("FinishEvent: Handler called")
	idStr := r.PathValue("id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		fmt.Printf("FinishEvent: Invalid event id: %s\n", idStr)
		http.Error(w, "Invalid event id", http.StatusBadRequest)
		return
	}

	adminIDStr := r.URL.Query().Get("admin_id")
	if adminIDStr == "" {
		fmt.Println("FinishEvent: Missing admin_id")
		http.Error(w, "Missing admin_id", http.StatusBadRequest)
		return
	}
	adminID, err := primitive.ObjectIDFromHex(adminIDStr)
	if err != nil {
		fmt.Printf("FinishEvent: Invalid admin_id: %s\n", adminIDStr)
		http.Error(w, "Invalid admin_id", http.StatusBadRequest)
		return
	}

	collection := s.DB.GetCollection("events")
	var event models.Event
	err = collection.FindOne(context.Background(), bson.M{"_id": id}).Decode(&event)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}
	// Verify admin of the group
	groupsCollection := s.DB.GetCollection("groups")
	var group models.Group
	err = groupsCollection.FindOne(context.Background(), bson.M{"_id": event.GroupID}).Decode(&group)
	if err != nil {
		fmt.Println("FinishEvent: Group not found")
		http.Error(w, "Group not found", http.StatusInternalServerError)
		return
	}

	fmt.Printf("FinishEvent: UserID=%s, GroupAdminID=%s, EventHostID=%s\n", adminID.Hex(), group.AdminID.Hex(), event.HostID.Hex())

	if group.AdminID != adminID && event.HostID != adminID {
		fmt.Println("FinishEvent: Unauthorized")
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	if event.Recurrence == "" {
		fmt.Println("FinishEvent: Event is not recurring")
		http.Error(w, "Event is not recurring", http.StatusBadRequest)
		return
	}

	// Create next event
	daysToAdd := 0
	switch event.Recurrence {
	case "Weekly":
		daysToAdd = 7
	case "Bi-Weekly":
		daysToAdd = 14
	}

	newEvent := event
	newEvent.ID = primitive.NewObjectID()
	newEvent.Date = event.Date.AddDate(0, 0, daysToAdd)
	newEvent.GuestIDs = []primitive.ObjectID{}  // Clear guest list
	newEvent.GuestJoinCode = generateJoinCode() // Generate new join code

	// Rotate Host
	fmt.Println("FinishEvent: Rotating host...")
	// 1. Fetch all group members
	familiesCollection := s.DB.GetCollection("families")
	cursor, err := familiesCollection.Find(context.Background(), bson.M{"group_ids": event.GroupID})
	if err == nil {
		var members []models.Family
		if err = cursor.All(context.Background(), &members); err == nil && len(members) > 0 {
			fmt.Printf("FinishEvent: Found %d members\n", len(members))
			// 2. Sort members by ID for deterministic rotation
			sort.Slice(members, func(i, j int) bool {
				return members[i].ID.Hex() < members[j].ID.Hex()
			})

			// 3. Find current host index
			currentIndex := -1
			for i, m := range members {
				if m.ID == event.HostID {
					currentIndex = i
					break
				}
			}
			fmt.Printf("FinishEvent: Current host index: %d\n", currentIndex)

			// 4. Select next host
			if currentIndex != -1 {
				nextIndex := (currentIndex + 1) % len(members)
				// Ensure not same host if possible (only if > 1 member)
				if len(members) > 1 && members[nextIndex].ID == event.HostID {
					// This shouldn't happen with simple modulo unless len=1, but just in case logic changes
					nextIndex = (nextIndex + 1) % len(members)
				}
				newEvent.HostID = members[nextIndex].ID
				fmt.Printf("FinishEvent: New host ID: %s\n", newEvent.HostID.Hex())
			} else {
				// Current host not found (maybe left group?), just pick first member
				newEvent.HostID = members[0].ID
				fmt.Printf("FinishEvent: Current host not found, defaulting to: %s\n", newEvent.HostID.Hex())
			}
		} else {
			fmt.Println("FinishEvent: No members found or error decoding")
		}
	} else {
		fmt.Printf("FinishEvent: Error finding members: %v\n", err)
	}

	_, err = collection.InsertOne(context.Background(), newEvent)
	if err != nil {
		fmt.Printf("FinishEvent: Failed to insert new event: %v\n", err)
		http.Error(w, "Failed to create next event", http.StatusInternalServerError)
		return
	}
	fmt.Println("FinishEvent: Successfully created next event")

	// Mark the old event as completed instead of deleting
	_, err = collection.UpdateOne(
		context.Background(),
		bson.M{"_id": id},
		bson.M{"$set": bson.M{"status": "completed"}},
	)
	if err != nil {
		fmt.Printf("Failed to mark old event as completed: %v\n", err)
	}

	// Broadcast update for new event
	msgNew := map[string]interface{}{
		"type": "event_created",
		"data": newEvent,
	}
	msgBytesNew, _ := json.Marshal(msgNew)
	s.Hub.Broadcast(msgBytesNew)

	// Broadcast deletion for old event (so it's removed from dashboard)
	msgDelete := map[string]interface{}{
		"type": "event_deleted",
		"data": map[string]interface{}{
			"event_id": id,
			"group_id": event.GroupID,
		},
	}
	msgBytesDelete, _ := json.Marshal(msgDelete)
	s.Hub.Broadcast(msgBytesDelete)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(newEvent)
}

func (s *Server) SkipEvent(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, "Invalid event id", http.StatusBadRequest)
		return
	}

	adminIDStr := r.URL.Query().Get("admin_id")
	if adminIDStr == "" {
		http.Error(w, "Missing admin_id", http.StatusBadRequest)
		return
	}
	adminID, err := primitive.ObjectIDFromHex(adminIDStr)
	if err != nil {
		http.Error(w, "Invalid admin_id", http.StatusBadRequest)
		return
	}

	collection := s.DB.GetCollection("events")
	var event models.Event
	err = collection.FindOne(context.Background(), bson.M{"_id": id}).Decode(&event)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	// Verify admin
	groupsCollection := s.DB.GetCollection("groups")
	var group models.Group
	err = groupsCollection.FindOne(context.Background(), bson.M{"_id": event.GroupID}).Decode(&group)
	if err != nil {
		http.Error(w, "Group not found", http.StatusInternalServerError)
		return
	}
	if group.AdminID != adminID && event.HostID != adminID {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	if event.Recurrence == "" {
		http.Error(w, "Event is not recurring", http.StatusBadRequest)
		return
	}

	// Update date
	daysToAdd := 0
	switch event.Recurrence {
	case "Weekly":
		daysToAdd = 7
	case "Bi-Weekly":
		daysToAdd = 14
	}

	newDate := event.Date.AddDate(0, 0, daysToAdd)
	_, err = collection.UpdateOne(
		context.Background(),
		bson.M{"_id": id},
		bson.M{"$set": bson.M{"date": newDate}},
	)
	if err != nil {
		http.Error(w, "Failed to update event", http.StatusInternalServerError)
		return
	}

	// Broadcast update
	event.Date = newDate
	msg := map[string]interface{}{
		"type": "event_updated",
		"data": event,
	}
	msgBytes, _ := json.Marshal(msg)
	s.Hub.Broadcast(msgBytes)

	w.WriteHeader(http.StatusOK)
}

func (s *Server) GetEvents(w http.ResponseWriter, r *http.Request) {
	groupIDStr := r.URL.Query().Get("group_id")
	if groupIDStr == "" {
		http.Error(w, "group_id is required", http.StatusBadRequest)
		return
	}

	groupID, err := primitive.ObjectIDFromHex(groupIDStr)
	if err != nil {
		http.Error(w, "invalid group_id", http.StatusBadRequest)
		return
	}

	collection := s.DB.GetCollection("events")
	filter := bson.M{
		"group_id": groupID,
		"status":   bson.M{"$ne": "completed"},
	}
	cursor, err := collection.Find(context.Background(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.Background())

	var events []models.Event
	if err = cursor.All(context.Background(), &events); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(events)
}

func (s *Server) GetUserEvents(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}

	collection := s.DB.GetCollection("events")

	// Find events where user is a guest
	filter := bson.M{
		"guest_ids": userID,
		"date":      bson.M{"$gte": time.Now().AddDate(0, 0, -1)}, // Only show future or recent events
	}

	cursor, err := collection.Find(context.Background(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.Background())

	var events []models.Event
	if err = cursor.All(context.Background(), &events); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(events)
}

func (s *Server) JoinEventByCode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FamilyID primitive.ObjectID `json:"family_id"`
		JoinCode string             `json:"join_code"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	collection := s.DB.GetCollection("events")
	var event models.Event
	err := collection.FindOne(context.Background(), bson.M{"guest_join_code": req.JoinCode}).Decode(&event)
	if err != nil {
		http.Error(w, "Invalid join code", http.StatusNotFound)
		return
	}

	// Check if event is in the past
	if event.Date.Before(time.Now().AddDate(0, 0, -1)) {
		http.Error(w, "Event has already finished", http.StatusForbidden)
		return
	}

	// Add user to guest_ids if not already present
	for _, id := range event.GuestIDs {
		if id == req.FamilyID {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(event)
			return
		}
	}

	_, err = collection.UpdateOne(
		context.Background(),
		bson.M{"_id": event.ID},
		bson.M{"$push": bson.M{"guest_ids": req.FamilyID}},
	)
	if err != nil {
		http.Error(w, "Failed to join event", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(event)
}

func (s *Server) GetEventByCode(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	collection := s.DB.GetCollection("events")
	var event models.Event
	err := collection.FindOne(context.Background(), bson.M{"guest_join_code": code}).Decode(&event)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	// Check if event is in the past
	if event.Date.Before(time.Now().AddDate(0, 0, -1)) {
		http.Error(w, "Event has already finished", http.StatusForbidden)
		return
	}

	// Fetch Host Name
	familiesCollection := s.DB.GetCollection("families")
	var host models.Family
	err = familiesCollection.FindOne(context.Background(), bson.M{"_id": event.HostID}).Decode(&host)
	if err == nil {
		event.HostName = host.Name
	}

	json.NewEncoder(w).Encode(event)
}

func (s *Server) GetEvent(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, "Invalid event id", http.StatusBadRequest)
		return
	}

	collection := s.DB.GetCollection("events")
	var event models.Event
	err = collection.FindOne(context.Background(), bson.M{"_id": id}).Decode(&event)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	// Fetch Host Name
	familiesCollection := s.DB.GetCollection("families")
	var host models.Family
	err = familiesCollection.FindOne(context.Background(), bson.M{"_id": event.HostID}).Decode(&host)
	if err == nil {
		event.HostName = host.Name
	}

	json.NewEncoder(w).Encode(event)
}

func (s *Server) DeleteEvent(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, "Invalid event id", http.StatusBadRequest)
		return
	}

	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		http.Error(w, "Missing user_id", http.StatusBadRequest)
		return
	}
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user_id", http.StatusBadRequest)
		return
	}

	collection := s.DB.GetCollection("events")
	var event models.Event
	err = collection.FindOne(context.Background(), bson.M{"_id": id}).Decode(&event)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	// Fetch group to check admin
	groupsCollection := s.DB.GetCollection("groups")
	var group models.Group
	err = groupsCollection.FindOne(context.Background(), bson.M{"_id": event.GroupID}).Decode(&group)
	if err != nil {
		http.Error(w, "Group not found", http.StatusInternalServerError)
		return
	}

	// Check authorization
	// Group Admin can delete any event
	// Event Host can only delete non-recurring events
	isAuthorized := false
	if group.AdminID == userID {
		isAuthorized = true
	} else if event.HostID == userID {
		if event.Recurrence == "" {
			isAuthorized = true
		}
	}

	if !isAuthorized {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	_, err = collection.DeleteOne(context.Background(), bson.M{"_id": id})
	if err != nil {
		http.Error(w, "Failed to delete event", http.StatusInternalServerError)
		return
	}

	// Broadcast update
	msg := map[string]interface{}{
		"type": "event_deleted",
		"data": map[string]interface{}{
			"event_id": id,
			"group_id": event.GroupID,
		},
	}
	msgBytes, _ := json.Marshal(msg)
	s.Hub.Broadcast(msgBytes)

	w.WriteHeader(http.StatusOK)
}

func (s *Server) GetEventStats(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, "Invalid event id", http.StatusBadRequest)
		return
	}

	collection := s.DB.GetCollection("events")
	var event models.Event
	err = collection.FindOne(context.Background(), bson.M{"_id": id}).Decode(&event)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	if event.RecurrenceID.IsZero() {
		// Not part of a recurrence series or legacy event
		json.NewEncoder(w).Encode(map[string]interface{}{
			"total_occurrences": 0,
			"host_counts":       []interface{}{},
		})
		return
	}

	// Find all completed events in this series
	filter := bson.M{
		"recurrence_id": event.RecurrenceID,
		"status":        "completed",
	}
	cursor, err := collection.Find(context.Background(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.Background())

	var completedEvents []models.Event
	if err = cursor.All(context.Background(), &completedEvents); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	totalOccurrences := len(completedEvents)
	hostCounts := make(map[string]int)

	for _, e := range completedEvents {
		hostCounts[e.HostID.Hex()]++
	}

	// Format for response
	type HostStat struct {
		FamilyID   string `json:"family_id"`
		FamilyName string `json:"family_name"`
		Count      int    `json:"count"`
	}
	var stats []HostStat

	familiesCollection := s.DB.GetCollection("families")
	for hostID, count := range hostCounts {
		hID, _ := primitive.ObjectIDFromHex(hostID)
		var family models.Family
		err := familiesCollection.FindOne(context.Background(), bson.M{"_id": hID}).Decode(&family)
		name := "Unknown"
		if err == nil {
			name = family.Name
		}
		stats = append(stats, HostStat{
			FamilyID:   hostID,
			FamilyName: name,
			Count:      count,
		})
	}

	// Sort stats by count descending
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Count > stats[j].Count
	})

	json.NewEncoder(w).Encode(map[string]interface{}{
		"total_occurrences": totalOccurrences,
		"host_counts":       stats,
	})
}
