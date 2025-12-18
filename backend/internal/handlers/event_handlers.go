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

	err := s.DB.CreateEvent(context.Background(), &event)
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

	event, err := s.DB.GetEvent(context.Background(), id)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	// Verify admin of the group
	group, err := s.DB.GetGroup(context.Background(), event.GroupID)
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

	// Create next event
	daysToAdd := 0
	switch event.Recurrence {
	case "Weekly":
		daysToAdd = 7
	case "Bi-Weekly":
		daysToAdd = 14
	}

	newEvent := *event
	newEvent.ID = primitive.NewObjectID()
	newEvent.Date = event.Date.AddDate(0, 0, daysToAdd)
	newEvent.GuestIDs = []primitive.ObjectID{}  // Clear guest list
	newEvent.GuestJoinCode = generateJoinCode() // Generate new join code

	// Rotate Host
	members, err := s.DB.GetFamiliesByGroupID(context.Background(), event.GroupID)
	if err == nil && len(members) > 0 {
		sort.Slice(members, func(i, j int) bool {
			return members[i].ID.Hex() < members[j].ID.Hex()
		})

		currentIndex := -1
		for i, m := range members {
			if m.ID == event.HostID {
				currentIndex = i
				break
			}
		}

		if currentIndex != -1 {
			nextIndex := (currentIndex + 1) % len(members)
			newEvent.HostID = members[nextIndex].ID
		} else {
			newEvent.HostID = members[0].ID
		}
	}

	err = s.DB.CreateEvent(context.Background(), &newEvent)
	if err != nil {
		http.Error(w, "Failed to create next event", http.StatusInternalServerError)
		return
	}

	// Mark the old event as completed
	err = s.DB.UpdateEvent(
		context.Background(),
		id,
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

	// Broadcast deletion for old event
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

	event, err := s.DB.GetEvent(context.Background(), id)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	// Verify admin
	group, err := s.DB.GetGroup(context.Background(), event.GroupID)
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
	err = s.DB.UpdateEvent(
		context.Background(),
		id,
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

	events, err := s.DB.GetEventsByGroupID(context.Background(), groupID, false)
	if err != nil {
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

	events, err := s.DB.GetEventsByUserID(context.Background(), userID)
	if err != nil {
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

	event, err := s.DB.GetEventByCode(context.Background(), req.JoinCode)
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

	err = s.DB.UpdateEvent(
		context.Background(),
		event.ID,
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
	event, err := s.DB.GetEventByCode(context.Background(), code)
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
	host, err := s.DB.GetFamilyByID(context.Background(), event.HostID)
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

	event, err := s.DB.GetEvent(context.Background(), id)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	// Fetch Host Name
	host, err := s.DB.GetFamilyByID(context.Background(), event.HostID)
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

	event, err := s.DB.GetEvent(context.Background(), id)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	// Fetch group to check admin
	group, err := s.DB.GetGroup(context.Background(), event.GroupID)
	if err != nil {
		http.Error(w, "Group not found", http.StatusInternalServerError)
		return
	}

	// Check authorization
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

	err = s.DB.DeleteEvent(context.Background(), id)
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

	event, err := s.DB.GetEvent(context.Background(), id)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	if event.RecurrenceID.IsZero() {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"total_occurrences": 0,
			"host_counts":       []interface{}{},
		})
		return
	}

	completedEvents, err := s.DB.GetCompletedEventsByRecurrenceID(context.Background(), event.RecurrenceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	totalOccurrences := len(completedEvents)
	hostCounts := make(map[string]int)

	for _, e := range completedEvents {
		hostCounts[e.HostID.Hex()]++
	}

	type HostStat struct {
		FamilyID   string `json:"family_id"`
		FamilyName string `json:"family_name"`
		Count      int    `json:"count"`
	}
	var stats []HostStat

	for hostID, count := range hostCounts {
		hID, _ := primitive.ObjectIDFromHex(hostID)
		family, err := s.DB.GetFamilyByID(context.Background(), hID)
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

	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Count > stats[j].Count
	})

	json.NewEncoder(w).Encode(map[string]interface{}{
		"total_occurrences": totalOccurrences,
		"host_counts":       stats,
	})
}
