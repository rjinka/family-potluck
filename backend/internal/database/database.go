package database

import (
	"context"
	"family-potluck/backend/internal/models"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var ErrNoDocuments = mongo.ErrNoDocuments

type Service interface {
	Health() map[string]string
	Close() error
	GetCollection(name string) *mongo.Collection

	// Families
	GetFamilyByEmail(ctx context.Context, email string) (*models.Family, error)
	GetFamilyByID(ctx context.Context, id primitive.ObjectID) (*models.Family, error)
	CreateFamily(ctx context.Context, family *models.Family) error
	UpdateFamily(ctx context.Context, id primitive.ObjectID, update bson.M) error
	GetFamiliesByGroupID(ctx context.Context, groupID primitive.ObjectID) ([]models.Family, error)
	GetFamiliesByIDs(ctx context.Context, ids []primitive.ObjectID) ([]models.Family, error)
	RemoveGroupIDFromAllFamilies(ctx context.Context, groupID primitive.ObjectID) error

	// Groups
	CountGroupsByName(ctx context.Context, name string) (int64, error)
	CreateGroup(ctx context.Context, group *models.Group) error
	GetGroup(ctx context.Context, id primitive.ObjectID) (*models.Group, error)
	GetGroupByCode(ctx context.Context, code string) (*models.Group, error)
	GetGroups(ctx context.Context) ([]models.Group, error)
	DeleteGroup(ctx context.Context, id primitive.ObjectID) error

	// Events
	CreateEvent(ctx context.Context, event *models.Event) error
	GetEvent(ctx context.Context, id primitive.ObjectID) (*models.Event, error)
	GetEventByCode(ctx context.Context, code string) (*models.Event, error)
	GetEventsByGroupID(ctx context.Context, groupID primitive.ObjectID, includeCompleted bool) ([]models.Event, error)
	GetEventsByUserID(ctx context.Context, userID primitive.ObjectID) ([]models.Event, error)
	UpdateEvent(ctx context.Context, id primitive.ObjectID, update bson.M) error
	DeleteEvent(ctx context.Context, id primitive.ObjectID) error
	GetCompletedEventsByRecurrenceID(ctx context.Context, recurrenceID primitive.ObjectID) ([]models.Event, error)

	// Dishes
	CreateDish(ctx context.Context, dish *models.Dish) error
	GetDishesByEventID(ctx context.Context, eventID primitive.ObjectID) ([]models.Dish, error)
	GetDishByID(ctx context.Context, id primitive.ObjectID) (*models.Dish, error)
	UpdateDish(ctx context.Context, id primitive.ObjectID, update bson.M) error
	DeleteDish(ctx context.Context, id primitive.ObjectID) error

	// RSVPs
	UpsertRSVP(ctx context.Context, rsvp *models.RSVP) (primitive.ObjectID, error)
	GetRSVPsByEventID(ctx context.Context, eventID primitive.ObjectID) ([]models.RSVP, error)

	// Swaps
	CreateSwapRequest(ctx context.Context, swap *models.SwapRequest) error
	GetSwapRequestsByEventID(ctx context.Context, eventID primitive.ObjectID) ([]models.SwapRequest, error)
	GetSwapRequestByID(ctx context.Context, id primitive.ObjectID) (*models.SwapRequest, error)
	UpdateSwapRequest(ctx context.Context, id primitive.ObjectID, update bson.M) error

	// Chat
	CreateChatMessage(ctx context.Context, msg *models.ChatMessage) error
	GetChatMessagesByEventID(ctx context.Context, eventID primitive.ObjectID) ([]models.ChatMessage, error)

	// Households
	CreateHousehold(ctx context.Context, household *models.Household) error
	GetHousehold(ctx context.Context, id primitive.ObjectID) (*models.Household, error)
	UpdateHousehold(ctx context.Context, id primitive.ObjectID, update bson.M) error
	DeleteHousehold(ctx context.Context, id primitive.ObjectID) error
	RemoveMemberFromHousehold(ctx context.Context, householdID, familyID primitive.ObjectID) error
}

type service struct {
	db *mongo.Database
}

func New() Service {
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal(err)
	}

	dbName := os.Getenv("MONGODB_DATABASE")
	if dbName == "" {
		dbName = "familypotluck"
	}

	dbInstance := &service{
		db: client.Database(dbName),
	}

	return dbInstance
}

func (s *service) Health() map[string]string {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := s.db.Client().Ping(ctx, nil)
	if err != nil {
		return map[string]string{
			"status":  "down",
			"error":   err.Error(),
			"message": "MongoDB is not responding",
		}
	}

	return map[string]string{
		"status":  "up",
		"message": "MongoDB is responding",
	}
}

func (s *service) Close() error {
	return s.db.Client().Disconnect(context.Background())
}

func (s *service) GetCollection(name string) *mongo.Collection {
	return s.db.Collection(name)
}

// Families implementation
func (s *service) GetFamilyByEmail(ctx context.Context, email string) (*models.Family, error) {
	var family models.Family
	err := s.db.Collection("families").FindOne(ctx, bson.M{"email": email}).Decode(&family)
	if err != nil {
		return nil, err
	}
	return &family, nil
}

func (s *service) GetFamilyByID(ctx context.Context, id primitive.ObjectID) (*models.Family, error) {
	var family models.Family
	err := s.db.Collection("families").FindOne(ctx, bson.M{"_id": id}).Decode(&family)
	if err != nil {
		return nil, err
	}
	return &family, nil
}

func (s *service) CreateFamily(ctx context.Context, family *models.Family) error {
	_, err := s.db.Collection("families").InsertOne(ctx, family)
	return err
}

func (s *service) UpdateFamily(ctx context.Context, id primitive.ObjectID, update bson.M) error {
	_, err := s.db.Collection("families").UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

func (s *service) GetFamiliesByGroupID(ctx context.Context, groupID primitive.ObjectID) ([]models.Family, error) {
	cursor, err := s.db.Collection("families").Find(ctx, bson.M{"group_ids": groupID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var families []models.Family
	if err = cursor.All(ctx, &families); err != nil {
		return nil, err
	}
	return families, nil
}

func (s *service) GetFamiliesByIDs(ctx context.Context, ids []primitive.ObjectID) ([]models.Family, error) {
	filter := bson.M{"_id": bson.M{"$in": ids}}
	cursor, err := s.db.Collection("families").Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var families []models.Family
	if err = cursor.All(ctx, &families); err != nil {
		return nil, err
	}
	return families, nil
}

func (s *service) RemoveGroupIDFromAllFamilies(ctx context.Context, groupID primitive.ObjectID) error {
	_, err := s.db.Collection("families").UpdateMany(
		ctx,
		bson.M{"group_ids": groupID},
		bson.M{"$pull": bson.M{"group_ids": groupID}},
	)
	return err
}

// Groups implementation
func (s *service) CountGroupsByName(ctx context.Context, name string) (int64, error) {
	return s.db.Collection("groups").CountDocuments(ctx, bson.M{"name": name})
}

func (s *service) CreateGroup(ctx context.Context, group *models.Group) error {
	_, err := s.db.Collection("groups").InsertOne(ctx, group)
	return err
}

func (s *service) GetGroup(ctx context.Context, id primitive.ObjectID) (*models.Group, error) {
	var group models.Group
	err := s.db.Collection("groups").FindOne(ctx, bson.M{"_id": id}).Decode(&group)
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func (s *service) GetGroupByCode(ctx context.Context, code string) (*models.Group, error) {
	var group models.Group
	err := s.db.Collection("groups").FindOne(ctx, bson.M{"join_code": code}).Decode(&group)
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func (s *service) GetGroups(ctx context.Context) ([]models.Group, error) {
	cursor, err := s.db.Collection("groups").Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var groups []models.Group
	if err = cursor.All(ctx, &groups); err != nil {
		return nil, err
	}
	return groups, nil
}

func (s *service) DeleteGroup(ctx context.Context, id primitive.ObjectID) error {
	_, err := s.db.Collection("groups").DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// Events implementation
func (s *service) CreateEvent(ctx context.Context, event *models.Event) error {
	_, err := s.db.Collection("events").InsertOne(ctx, event)
	return err
}

func (s *service) GetEvent(ctx context.Context, id primitive.ObjectID) (*models.Event, error) {
	var event models.Event
	err := s.db.Collection("events").FindOne(ctx, bson.M{"_id": id}).Decode(&event)
	if err != nil {
		return nil, err
	}
	return &event, nil
}

func (s *service) GetEventByCode(ctx context.Context, code string) (*models.Event, error) {
	var event models.Event
	err := s.db.Collection("events").FindOne(ctx, bson.M{"guest_join_code": code}).Decode(&event)
	if err != nil {
		return nil, err
	}
	return &event, nil
}

func (s *service) GetEventsByGroupID(ctx context.Context, groupID primitive.ObjectID, includeCompleted bool) ([]models.Event, error) {
	filter := bson.M{"group_id": groupID}
	if !includeCompleted {
		filter["status"] = bson.M{"$ne": "completed"}
	}
	cursor, err := s.db.Collection("events").Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var events []models.Event
	if err = cursor.All(ctx, &events); err != nil {
		return nil, err
	}
	return events, nil
}

func (s *service) GetEventsByUserID(ctx context.Context, userID primitive.ObjectID) ([]models.Event, error) {
	filter := bson.M{
		"guest_ids": userID,
		"date":      bson.M{"$gte": time.Now().AddDate(0, 0, -1)},
	}
	cursor, err := s.db.Collection("events").Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var events []models.Event
	if err = cursor.All(ctx, &events); err != nil {
		return nil, err
	}
	return events, nil
}

func (s *service) UpdateEvent(ctx context.Context, id primitive.ObjectID, update bson.M) error {
	_, err := s.db.Collection("events").UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

func (s *service) DeleteEvent(ctx context.Context, id primitive.ObjectID) error {
	_, err := s.db.Collection("events").DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (s *service) GetCompletedEventsByRecurrenceID(ctx context.Context, recurrenceID primitive.ObjectID) ([]models.Event, error) {
	filter := bson.M{
		"recurrence_id": recurrenceID,
		"status":        "completed",
	}
	cursor, err := s.db.Collection("events").Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var events []models.Event
	if err = cursor.All(ctx, &events); err != nil {
		return nil, err
	}
	return events, nil
}

// Dishes implementation
func (s *service) CreateDish(ctx context.Context, dish *models.Dish) error {
	_, err := s.db.Collection("dishes").InsertOne(ctx, dish)
	return err
}

func (s *service) GetDishesByEventID(ctx context.Context, eventID primitive.ObjectID) ([]models.Dish, error) {
	cursor, err := s.db.Collection("dishes").Find(ctx, bson.M{"event_id": eventID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var dishes []models.Dish
	if err = cursor.All(ctx, &dishes); err != nil {
		return nil, err
	}
	return dishes, nil
}

func (s *service) GetDishByID(ctx context.Context, id primitive.ObjectID) (*models.Dish, error) {
	var dish models.Dish
	err := s.db.Collection("dishes").FindOne(ctx, bson.M{"_id": id}).Decode(&dish)
	if err != nil {
		return nil, err
	}
	return &dish, nil
}

func (s *service) UpdateDish(ctx context.Context, id primitive.ObjectID, update bson.M) error {
	_, err := s.db.Collection("dishes").UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

func (s *service) DeleteDish(ctx context.Context, id primitive.ObjectID) error {
	_, err := s.db.Collection("dishes").DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// RSVPs implementation
func (s *service) UpsertRSVP(ctx context.Context, rsvp *models.RSVP) (primitive.ObjectID, error) {
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

	result, err := s.db.Collection("rsvps").UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return primitive.NilObjectID, err
	}

	if result.UpsertedID != nil {
		return result.UpsertedID.(primitive.ObjectID), nil
	}

	var existing models.RSVP
	err = s.db.Collection("rsvps").FindOne(ctx, filter).Decode(&existing)
	if err != nil {
		return primitive.NilObjectID, err
	}
	return existing.ID, nil
}

func (s *service) GetRSVPsByEventID(ctx context.Context, eventID primitive.ObjectID) ([]models.RSVP, error) {
	cursor, err := s.db.Collection("rsvps").Find(ctx, bson.M{"event_id": eventID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var rsvps []models.RSVP
	if err = cursor.All(ctx, &rsvps); err != nil {
		return nil, err
	}
	return rsvps, nil
}

// Swaps implementation
func (s *service) CreateSwapRequest(ctx context.Context, swap *models.SwapRequest) error {
	_, err := s.db.Collection("swaps").InsertOne(ctx, swap)
	return err
}

func (s *service) GetSwapRequestsByEventID(ctx context.Context, eventID primitive.ObjectID) ([]models.SwapRequest, error) {
	cursor, err := s.db.Collection("swaps").Find(ctx, bson.M{"event_id": eventID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var swaps []models.SwapRequest
	if err = cursor.All(ctx, &swaps); err != nil {
		return nil, err
	}
	return swaps, nil
}

func (s *service) GetSwapRequestByID(ctx context.Context, id primitive.ObjectID) (*models.SwapRequest, error) {
	var swap models.SwapRequest
	err := s.db.Collection("swaps").FindOne(ctx, bson.M{"_id": id}).Decode(&swap)
	if err != nil {
		return nil, err
	}
	return &swap, nil
}

func (s *service) UpdateSwapRequest(ctx context.Context, id primitive.ObjectID, update bson.M) error {
	_, err := s.db.Collection("swaps").UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

// Chat implementation
func (s *service) CreateChatMessage(ctx context.Context, msg *models.ChatMessage) error {
	_, err := s.db.Collection("chat").InsertOne(ctx, msg)
	return err
}

func (s *service) GetChatMessagesByEventID(ctx context.Context, eventID primitive.ObjectID) ([]models.ChatMessage, error) {
	cursor, err := s.db.Collection("chat").Find(ctx, bson.M{"event_id": eventID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var msgs []models.ChatMessage
	if err = cursor.All(ctx, &msgs); err != nil {
		return nil, err
	}
	return msgs, nil
}

// Households implementation
func (s *service) CreateHousehold(ctx context.Context, household *models.Household) error {
	_, err := s.db.Collection("households").InsertOne(ctx, household)
	return err
}

func (s *service) GetHousehold(ctx context.Context, id primitive.ObjectID) (*models.Household, error) {
	var household models.Household
	err := s.db.Collection("households").FindOne(ctx, bson.M{"_id": id}).Decode(&household)
	if err != nil {
		return nil, err
	}
	return &household, nil
}

func (s *service) UpdateHousehold(ctx context.Context, id primitive.ObjectID, update bson.M) error {
	_, err := s.db.Collection("households").UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}
