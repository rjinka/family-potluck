package handlers

import (
	"context"
	"encoding/json"
	"family-potluck/backend/internal/database"
	"family-potluck/backend/internal/models"
	"family-potluck/backend/internal/websocket"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/api/idtoken"
)

type Server struct {
	DB  database.Service
	Hub *websocket.Hub
}

func NewServer(db database.Service, hub *websocket.Hub) *Server {
	return &Server{
		DB:  db,
		Hub: hub,
	}
}

func (s *Server) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDToken string `json:"id_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	// Ideally, validate the audience (your Client ID)
	payload, err := idtoken.Validate(ctx, req.IDToken, os.Getenv("GOOGLE_CLIENT_ID"))
	if err != nil {
		http.Error(w, "Invalid token: "+err.Error(), http.StatusUnauthorized)
		return
	}

	email := payload.Claims["email"].(string)
	name := payload.Claims["name"].(string)
	picture := payload.Claims["picture"].(string)
	sub := payload.Subject // Google ID

	collection := s.DB.GetCollection("families")
	var family models.Family
	err = collection.FindOne(ctx, bson.M{"email": email}).Decode(&family)

	if err == mongo.ErrNoDocuments {
		// Create new user
		family = models.Family{
			ID:       primitive.NewObjectID(),
			Name:     name,
			Email:    email,
			GoogleID: sub,
			Picture:  picture,
		}
		_, err = collection.InsertOne(ctx, family)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else {
		// Update existing user info if needed
		update := bson.M{
			"$set": bson.M{
				"name":      name,
				"picture":   picture,
				"google_id": sub,
			},
		}
		_, err = collection.UpdateOne(ctx, bson.M{"_id": family.ID}, update)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Create a session or JWT here if needed. For now, we'll just return the family object.
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(family)
}

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

func (s *Server) RSVPEvent(w http.ResponseWriter, r *http.Request) {
	var rsvp models.RSVP
	if err := json.NewDecoder(r.Body).Decode(&rsvp); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	collection := s.DB.GetCollection("rsvps")

	// Upsert RSVP
	filter := bson.M{
		"event_id":  rsvp.EventID,
		"family_id": rsvp.FamilyID,
	}
	update := bson.M{
		"$set": bson.M{
			"status":     rsvp.Status,
			"count":      rsvp.Count,
			"kids_count": rsvp.KidsCount,
		},
	}
	opts := options.Update().SetUpsert(true)

	result, err := collection.UpdateOne(context.Background(), filter, update, opts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// If upserted, we need to get the ID. If updated, we keep existing ID (not critical for frontend usually)
	// For simplicity, we can just return the input RSVP with success status
	if result.UpsertedID != nil {
		rsvp.ID = result.UpsertedID.(primitive.ObjectID)
	}

	// Broadcast update
	msg := map[string]interface{}{
		"type": "rsvp_updated",
		"data": rsvp,
	}
	msgBytes, _ := json.Marshal(msg)
	s.Hub.Broadcast(msgBytes)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(rsvp)
}

func (s *Server) GetRSVPs(w http.ResponseWriter, r *http.Request) {
	eventIDStr := r.URL.Query().Get("event_id")
	if eventIDStr == "" {
		http.Error(w, "event_id is required", http.StatusBadRequest)
		return
	}

	eventID, err := primitive.ObjectIDFromHex(eventIDStr)
	if err != nil {
		http.Error(w, "invalid event_id", http.StatusBadRequest)
		return
	}

	collection := s.DB.GetCollection("rsvps")
	cursor, err := collection.Find(context.Background(), bson.M{"event_id": eventID})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.Background())

	var rsvps []models.RSVP
	if err = cursor.All(context.Background(), &rsvps); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Populate Family details
	familiesCollection := s.DB.GetCollection("families")
	for i := range rsvps {
		var family models.Family
		err := familiesCollection.FindOne(context.Background(), bson.M{"_id": rsvps[i].FamilyID}).Decode(&family)
		if err == nil {
			rsvps[i].FamilyName = family.Name
			rsvps[i].FamilyPicture = family.Picture
		}
	}

	json.NewEncoder(w).Encode(rsvps)
}

func (s *Server) GetGroupMembers(w http.ResponseWriter, r *http.Request) {
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

	collection := s.DB.GetCollection("families")
	cursor, err := collection.Find(context.Background(), bson.M{"group_ids": groupID})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.Background())

	var families []models.Family
	if err = cursor.All(context.Background(), &families); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(families)
}

func (s *Server) AddDish(w http.ResponseWriter, r *http.Request) {
	var dish models.Dish
	if err := json.NewDecoder(r.Body).Decode(&dish); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dish.ID = primitive.NewObjectID()
	collection := s.DB.GetCollection("dishes")
	_, err := collection.InsertOne(context.Background(), dish)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Broadcast update
	msg := map[string]interface{}{
		"type": "dish_added",
		"data": dish,
	}
	msgBytes, _ := json.Marshal(msg)
	s.Hub.Broadcast(msgBytes)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(dish)
}

func (s *Server) GetDishes(w http.ResponseWriter, r *http.Request) {
	eventIDStr := r.URL.Query().Get("event_id")
	if eventIDStr == "" {
		http.Error(w, "event_id is required", http.StatusBadRequest)
		return
	}

	eventID, err := primitive.ObjectIDFromHex(eventIDStr)
	if err != nil {
		http.Error(w, "invalid event_id", http.StatusBadRequest)
		return
	}

	collection := s.DB.GetCollection("dishes")
	cursor, err := collection.Find(context.Background(), bson.M{"event_id": eventID})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.Background())

	var dishes []models.Dish
	if err = cursor.All(context.Background(), &dishes); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Collect bringer IDs
	bringerIDs := []primitive.ObjectID{}
	for _, dish := range dishes {
		if dish.BringerID != nil {
			bringerIDs = append(bringerIDs, *dish.BringerID)
		}
	}

	// Fetch families if there are any bringers
	bringerNames := make(map[primitive.ObjectID]string)
	if len(bringerIDs) > 0 {
		familiesCollection := s.DB.GetCollection("families")
		filter := bson.M{"_id": bson.M{"$in": bringerIDs}}
		cursor, err := familiesCollection.Find(context.Background(), filter)
		if err == nil {
			defer cursor.Close(context.Background())
			var families []models.Family
			if err = cursor.All(context.Background(), &families); err == nil {
				for _, family := range families {
					bringerNames[family.ID] = family.Name
				}
			}
		}
	}

	// Populate BringerName
	for i := range dishes {
		if dishes[i].BringerID != nil {
			if name, ok := bringerNames[*dishes[i].BringerID]; ok {
				dishes[i].BringerName = name
			}
		}
	}

	json.NewEncoder(w).Encode(dishes)
}

func (s *Server) PledgeDish(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, "Invalid dish id", http.StatusBadRequest)
		return
	}

	var req struct {
		FamilyID primitive.ObjectID `json:"family_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	collection := s.DB.GetCollection("dishes")

	// Fetch dish to get event_id
	var dish models.Dish
	err = collection.FindOne(context.Background(), bson.M{"_id": id}).Decode(&dish)
	if err != nil {
		http.Error(w, "Dish not found", http.StatusNotFound)
		return
	}

	_, err = collection.UpdateOne(
		context.Background(),
		bson.M{"_id": id},
		bson.M{"$set": bson.M{"bringer_id": req.FamilyID}},
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Broadcast update
	msg := map[string]interface{}{
		"type": "dish_pledged",
		"data": map[string]interface{}{
			"dish_id":    id,
			"event_id":   dish.EventID,
			"bringer_id": req.FamilyID,
		},
	}
	msgBytes, _ := json.Marshal(msg)
	s.Hub.Broadcast(msgBytes)

	w.WriteHeader(http.StatusOK)
}

func (s *Server) UnpledgeDish(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, "Invalid dish id", http.StatusBadRequest)
		return
	}

	collection := s.DB.GetCollection("dishes")

	// Fetch dish to get event_id
	var dish models.Dish
	err = collection.FindOne(context.Background(), bson.M{"_id": id}).Decode(&dish)
	if err != nil {
		http.Error(w, "Dish not found", http.StatusNotFound)
		return
	}

	_, err = collection.UpdateOne(
		context.Background(),
		bson.M{"_id": id},
		bson.M{"$unset": bson.M{"bringer_id": ""}},
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Broadcast update
	msg := map[string]interface{}{
		"type": "dish_unpledged",
		"data": map[string]interface{}{
			"dish_id":  id,
			"event_id": dish.EventID,
		},
	}
	msgBytes, _ := json.Marshal(msg)
	s.Hub.Broadcast(msgBytes)

	w.WriteHeader(http.StatusOK)
}

func (s *Server) DeleteDish(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, "Invalid dish id", http.StatusBadRequest)
		return
	}

	collection := s.DB.GetCollection("dishes")

	// Fetch dish to get event_id
	var dish models.Dish
	err = collection.FindOne(context.Background(), bson.M{"_id": id}).Decode(&dish)
	if err != nil {
		http.Error(w, "Dish not found", http.StatusNotFound)
		return
	}

	_, err = collection.DeleteOne(context.Background(), bson.M{"_id": id})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Broadcast update
	msg := map[string]interface{}{
		"type": "dish_deleted",
		"data": map[string]interface{}{
			"dish_id":  id,
			"event_id": dish.EventID,
		},
	}
	msgBytes, _ := json.Marshal(msg)
	s.Hub.Broadcast(msgBytes)

	w.WriteHeader(http.StatusOK)
}

func (s *Server) CreateSwapRequest(w http.ResponseWriter, r *http.Request) {
	var req models.SwapRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	req.ID = primitive.NewObjectID()
	req.Status = "pending"
	req.CreatedAt = time.Now()

	collection := s.DB.GetCollection("swap_requests")
	_, err := collection.InsertOne(context.Background(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Broadcast update
	msg := map[string]interface{}{
		"type": "swap_created",
		"data": req,
	}
	msgBytes, _ := json.Marshal(msg)
	s.Hub.Broadcast(msgBytes)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(req)
}

func (s *Server) UpdateSwapRequest(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var update struct {
		Status         string              `json:"status"`
		TargetFamilyID *primitive.ObjectID `json:"target_family_id"`
		EventUpdates   *struct {
			Date     time.Time `json:"date"`
			Location string    `json:"location"`
		} `json:"event_updates"`
	}
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	collection := s.DB.GetCollection("swap_requests")

	// Fetch request to get details for broadcast
	var req models.SwapRequest
	err = collection.FindOne(context.Background(), bson.M{"_id": id}).Decode(&req)
	if err != nil {
		http.Error(w, "Swap request not found", http.StatusNotFound)
		return
	}

	updateFields := bson.M{"status": update.Status}
	if update.TargetFamilyID != nil {
		updateFields["target_family_id"] = update.TargetFamilyID
		req.TargetFamilyID = update.TargetFamilyID // Update local object for broadcast
	}

	_, err = collection.UpdateOne(
		context.Background(),
		bson.M{"_id": id},
		bson.M{"$set": updateFields},
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update the request object with new status for broadcast
	req.Status = update.Status

	// If approved, update the dish bringer or event host
	if req.Status == "approved" {
		if req.Type == "host" {
			// Update Event Host
			eventsCollection := s.DB.GetCollection("events")

			// Let's check if we have a TargetID.
			var newHostID primitive.ObjectID
			if req.TargetFamilyID != nil {
				newHostID = *req.TargetFamilyID
			}

			eventUpdateDoc := bson.M{"host_id": newHostID}
			if update.EventUpdates != nil {
				if !update.EventUpdates.Date.IsZero() {
					eventUpdateDoc["date"] = update.EventUpdates.Date
				}
				if update.EventUpdates.Location != "" {
					eventUpdateDoc["location"] = update.EventUpdates.Location
				}
			}

			_, err = eventsCollection.UpdateOne(
				context.Background(),
				bson.M{"_id": req.EventID},
				bson.M{"$set": eventUpdateDoc},
			)
			if err != nil {
				fmt.Printf("Failed to update event host: %v\n", err)
			} else {
				// Broadcast event update
				var updatedEvent models.Event
				eventsCollection.FindOne(context.Background(), bson.M{"_id": req.EventID}).Decode(&updatedEvent)

				eventMsg := map[string]interface{}{
					"type": "event_updated",
					"data": updatedEvent,
				}
				eventMsgBytes, _ := json.Marshal(eventMsg)
				s.Hub.Broadcast(eventMsgBytes)
			}

		} else {
			// Default to dish swap
			dishesCollection := s.DB.GetCollection("dishes")

			// Fetch dish to check current bringer to determine direction (Request vs Offer)
			var currentDish models.Dish
			err = dishesCollection.FindOne(context.Background(), bson.M{"_id": req.DishID}).Decode(&currentDish)

			var newBringerID primitive.ObjectID
			if err == nil && currentDish.BringerID != nil && *currentDish.BringerID == req.RequestingFamilyID {
				// Requester is the current bringer -> It's an Offer -> Target (Acceptor) becomes bringer
				if req.TargetFamilyID != nil {
					newBringerID = *req.TargetFamilyID
				} else {
					// Fallback or error handling if needed, but TargetFamilyID should be set on acceptance
					newBringerID = req.RequestingFamilyID // Revert to requester if no target? Or fail?
				}
			} else {
				// Requester is NOT the current bringer -> It's a Request -> Requester becomes bringer
				newBringerID = req.RequestingFamilyID
			}

			_, err = dishesCollection.UpdateOne(
				context.Background(),
				bson.M{"_id": req.DishID},
				bson.M{"$set": bson.M{"bringer_id": newBringerID}},
			)
			if err != nil {
				// Log error but continue
				fmt.Printf("Failed to update dish bringer: %v\n", err)
			} else {
				// Broadcast dish update
				dishMsg := map[string]interface{}{
					"type": "dish_pledged",
					"data": map[string]interface{}{
						"dish_id":    req.DishID,
						"event_id":   req.EventID,
						"bringer_id": req.RequestingFamilyID,
					},
				}
				dishMsgBytes, _ := json.Marshal(dishMsg)
				s.Hub.Broadcast(dishMsgBytes)
			}
		}
	}

	// Broadcast update
	msg := map[string]interface{}{
		"type": "swap_updated",
		"data": req,
	}
	msgBytes, _ := json.Marshal(msg)
	s.Hub.Broadcast(msgBytes)

	w.WriteHeader(http.StatusOK)
}

func (s *Server) GetSwapRequests(w http.ResponseWriter, r *http.Request) {
	eventIDStr := r.URL.Query().Get("event_id")
	if eventIDStr == "" {
		http.Error(w, "event_id is required", http.StatusBadRequest)
		return
	}

	eventID, err := primitive.ObjectIDFromHex(eventIDStr)
	if err != nil {
		http.Error(w, "invalid event_id", http.StatusBadRequest)
		return
	}

	collection := s.DB.GetCollection("swap_requests")
	cursor, err := collection.Find(context.Background(), bson.M{"event_id": eventID})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.Background())

	var requests []models.SwapRequest
	if err = cursor.All(context.Background(), &requests); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Populate family names
	familiesCollection := s.DB.GetCollection("families")
	for i := range requests {
		var requestingFamily models.Family
		err := familiesCollection.FindOne(context.Background(), bson.M{"_id": requests[i].RequestingFamilyID}).Decode(&requestingFamily)
		if err == nil {
			requests[i].RequestingFamilyName = requestingFamily.Name
		}

		if requests[i].TargetFamilyID != nil {
			var targetFamily models.Family
			err := familiesCollection.FindOne(context.Background(), bson.M{"_id": requests[i].TargetFamilyID}).Decode(&targetFamily)
			if err == nil {
				requests[i].TargetFamilyName = targetFamily.Name
			}
		}
	}

	json.NewEncoder(w).Encode(requests)
}

func (s *Server) GetFamily(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "Missing id parameter", http.StatusBadRequest)
		return
	}

	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, "Invalid id format", http.StatusBadRequest)
		return
	}

	collection := s.DB.GetCollection("families")
	var family models.Family
	err = collection.FindOne(context.Background(), bson.M{"_id": id}).Decode(&family)
	if err != nil {
		http.Error(w, "Family not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(family)
}

func (s *Server) CreateGroup(w http.ResponseWriter, r *http.Request) {
	var group models.Group
	if err := json.NewDecoder(r.Body).Decode(&group); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check if group name already exists
	collection := s.DB.GetCollection("groups")
	count, err := collection.CountDocuments(context.Background(), bson.M{"name": group.Name})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if count > 0 {
		http.Error(w, "Group name already exists", http.StatusConflict)
		return
	}

	group.ID = primitive.NewObjectID()
	group.JoinCode = generateJoinCode()
	_, err = collection.InsertOne(context.Background(), group)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update admin's family to join this group
	familiesCollection := s.DB.GetCollection("families")
	_, err = familiesCollection.UpdateOne(
		context.Background(),
		bson.M{"_id": group.AdminID},
		bson.M{"$push": bson.M{"group_ids": group.ID}},
	)
	if err != nil {
		// Log error but don't fail the request as group is created
		// Ideally we should use a transaction here
		http.Error(w, "Group created but failed to update admin membership", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(group)
}

func (s *Server) JoinGroup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FamilyID primitive.ObjectID `json:"family_id"`
		GroupID  primitive.ObjectID `json:"group_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	familiesCollection := s.DB.GetCollection("families")
	_, err := familiesCollection.UpdateOne(
		context.Background(),
		bson.M{"_id": req.FamilyID},
		bson.M{"$addToSet": bson.M{"group_ids": req.GroupID}},
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) LeaveGroup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FamilyID primitive.ObjectID `json:"family_id"`
		GroupID  primitive.ObjectID `json:"group_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Prevent admin from leaving (they must delete the group)
	groupsCollection := s.DB.GetCollection("groups")
	var group models.Group
	err := groupsCollection.FindOne(context.Background(), bson.M{"_id": req.GroupID}).Decode(&group)
	if err != nil {
		http.Error(w, "Group not found", http.StatusNotFound)
		return
	}

	if group.AdminID == req.FamilyID {
		http.Error(w, "Admin cannot leave the group. Delete the group instead.", http.StatusForbidden)
		return
	}

	familiesCollection := s.DB.GetCollection("families")
	_, err = familiesCollection.UpdateOne(
		context.Background(),
		bson.M{"_id": req.FamilyID},
		bson.M{"$pull": bson.M{"group_ids": req.GroupID}},
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) JoinGroupByCode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FamilyID primitive.ObjectID `json:"family_id"`
		JoinCode string             `json:"join_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Find group by join code
	collection := s.DB.GetCollection("groups")
	var group models.Group
	err := collection.FindOne(context.Background(), bson.M{"join_code": req.JoinCode}).Decode(&group)
	if err != nil {
		http.Error(w, "Invalid join code", http.StatusNotFound)
		return
	}

	familiesCollection := s.DB.GetCollection("families")
	_, err = familiesCollection.UpdateOne(
		context.Background(),
		bson.M{"_id": req.FamilyID},
		bson.M{"$addToSet": bson.M{"group_ids": group.ID}},
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(group)
}

func (s *Server) GetGroupByCode(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	if code == "" {
		http.Error(w, "Missing join code", http.StatusBadRequest)
		return
	}

	collection := s.DB.GetCollection("groups")
	var group models.Group
	err := collection.FindOne(context.Background(), bson.M{"join_code": code}).Decode(&group)
	if err != nil {
		http.Error(w, "Group not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(group)
}

func generateJoinCode() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, 6)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func (s *Server) GetGroups(w http.ResponseWriter, r *http.Request) {
	collection := s.DB.GetCollection("groups")
	cursor, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.Background())

	var groups []models.Group
	if err = cursor.All(context.Background(), &groups); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(groups)
}

func (s *Server) GetGroup(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, "Invalid group id", http.StatusBadRequest)
		return
	}

	collection := s.DB.GetCollection("groups")
	var group models.Group
	err = collection.FindOne(context.Background(), bson.M{"_id": id}).Decode(&group)
	if err != nil {
		http.Error(w, "Group not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(group)
}

func (s *Server) DeleteGroup(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, "Invalid group id", http.StatusBadRequest)
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

	// Verify group exists and user is admin
	groupsCollection := s.DB.GetCollection("groups")
	var group models.Group
	err = groupsCollection.FindOne(context.Background(), bson.M{"_id": id}).Decode(&group)
	if err != nil {
		http.Error(w, "Group not found", http.StatusNotFound)
		return
	}

	if group.AdminID != adminID {
		http.Error(w, "Unauthorized: Only admin can delete group", http.StatusForbidden)
		return
	}

	// Delete group
	_, err = groupsCollection.DeleteOne(context.Background(), bson.M{"_id": id})
	if err != nil {
		http.Error(w, "Failed to delete group", http.StatusInternalServerError)
		return
	}

	// Remove group_id from all families
	familiesCollection := s.DB.GetCollection("families")
	_, err = familiesCollection.UpdateMany(
		context.Background(),
		bson.M{"group_ids": id},
		bson.M{"$pull": bson.M{"group_ids": id}},
	)
	if err != nil {
		// Log error but success for group deletion
		fmt.Printf("Failed to remove group from families: %v\n", err)
	}

	w.WriteHeader(http.StatusOK)
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

func (s *Server) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
