package database

import (
	"context"
	"family-potluck/backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type MockService struct {
	HealthFunc                            func() map[string]string
	CloseFunc                             func() error
	GetCollectionFunc                     func(name string) *mongo.Collection
	GetFamilyMemberByEmailFunc            func(ctx context.Context, email string) (*models.FamilyMember, error)
	GetFamilyMemberByIDFunc               func(ctx context.Context, id primitive.ObjectID) (*models.FamilyMember, error)
	CreateFamilyMemberFunc                func(ctx context.Context, familyMember *models.FamilyMember) error
	UpdateFamilyMemberFunc                func(ctx context.Context, id primitive.ObjectID, update bson.M) error
	GetFamilyMembersByGroupIDFunc         func(ctx context.Context, groupID primitive.ObjectID) ([]models.FamilyMember, error)
	GetFamilyMembersByIDsFunc             func(ctx context.Context, ids []primitive.ObjectID) ([]models.FamilyMember, error)
	RemoveGroupIDFromAllFamilyMembersFunc func(ctx context.Context, groupID primitive.ObjectID) error
	CountGroupsByNameFunc                 func(ctx context.Context, name string) (int64, error)
	CreateGroupFunc                       func(ctx context.Context, group *models.Group) error
	GetGroupFunc                          func(ctx context.Context, id primitive.ObjectID) (*models.Group, error)
	GetGroupByCodeFunc                    func(ctx context.Context, code string) (*models.Group, error)
	GetGroupsFunc                         func(ctx context.Context) ([]models.Group, error)
	UpdateGroupFunc                       func(ctx context.Context, id primitive.ObjectID, update bson.M) error
	DeleteGroupFunc                       func(ctx context.Context, id primitive.ObjectID) error
	CreateEventFunc                       func(ctx context.Context, event *models.Event) error
	GetEventFunc                          func(ctx context.Context, id primitive.ObjectID) (*models.Event, error)
	GetEventByCodeFunc                    func(ctx context.Context, code string) (*models.Event, error)
	GetEventsByGroupIDFunc                func(ctx context.Context, groupID primitive.ObjectID, includeCompleted bool) ([]models.Event, error)
	GetEventsByUserIDFunc                 func(ctx context.Context, userID primitive.ObjectID) ([]models.Event, error)
	UpdateEventFunc                       func(ctx context.Context, id primitive.ObjectID, update bson.M) error
	DeleteEventFunc                       func(ctx context.Context, id primitive.ObjectID) error
	GetCompletedEventsByRecurrenceIDFunc  func(ctx context.Context, recurrenceID primitive.ObjectID) ([]models.Event, error)
	CreateDishFunc                        func(ctx context.Context, dish *models.Dish) error
	GetDishesByEventIDFunc                func(ctx context.Context, eventID primitive.ObjectID) ([]models.Dish, error)
	GetDishByIDFunc                       func(ctx context.Context, id primitive.ObjectID) (*models.Dish, error)
	UpdateDishFunc                        func(ctx context.Context, id primitive.ObjectID, update bson.M) error
	DeleteDishFunc                        func(ctx context.Context, id primitive.ObjectID) error
	UpsertRSVPFunc                        func(ctx context.Context, rsvp *models.RSVP) (primitive.ObjectID, error)
	GetRSVPsByEventIDFunc                 func(ctx context.Context, eventID primitive.ObjectID) ([]models.RSVP, error)
	CreateSwapRequestFunc                 func(ctx context.Context, swap *models.SwapRequest) error
	GetSwapRequestsByEventIDFunc          func(ctx context.Context, eventID primitive.ObjectID) ([]models.SwapRequest, error)
	GetSwapRequestByIDFunc                func(ctx context.Context, id primitive.ObjectID) (*models.SwapRequest, error)
	UpdateSwapRequestFunc                 func(ctx context.Context, id primitive.ObjectID, update bson.M) error
	CreateChatMessageFunc                 func(ctx context.Context, msg *models.ChatMessage) error
	GetChatMessagesByEventIDFunc          func(ctx context.Context, eventID primitive.ObjectID) ([]models.ChatMessage, error)
	CreateHouseholdFunc                   func(ctx context.Context, household *models.Household) error
	GetHouseholdFunc                      func(ctx context.Context, id primitive.ObjectID) (*models.Household, error)
	UpdateHouseholdFunc                   func(ctx context.Context, id primitive.ObjectID, update bson.M) error
	DeleteHouseholdFunc                   func(ctx context.Context, id primitive.ObjectID) error
	RemoveMemberFromHouseholdFunc         func(ctx context.Context, householdID, familyID primitive.ObjectID) error
}

func (m *MockService) Health() map[string]string { return m.HealthFunc() }
func (m *MockService) Close() error              { return m.CloseFunc() }
func (m *MockService) GetCollection(name string) *mongo.Collection {
	return m.GetCollectionFunc(name)
}
func (m *MockService) GetFamilyMemberByEmail(ctx context.Context, email string) (*models.FamilyMember, error) {
	return m.GetFamilyMemberByEmailFunc(ctx, email)
}
func (m *MockService) GetFamilyMemberByID(ctx context.Context, id primitive.ObjectID) (*models.FamilyMember, error) {
	return m.GetFamilyMemberByIDFunc(ctx, id)
}
func (m *MockService) CreateFamilyMember(ctx context.Context, familyMember *models.FamilyMember) error {
	return m.CreateFamilyMemberFunc(ctx, familyMember)
}
func (m *MockService) UpdateFamilyMember(ctx context.Context, id primitive.ObjectID, update bson.M) error {
	return m.UpdateFamilyMemberFunc(ctx, id, update)
}
func (m *MockService) GetFamilyMembersByGroupID(ctx context.Context, groupID primitive.ObjectID) ([]models.FamilyMember, error) {
	return m.GetFamilyMembersByGroupIDFunc(ctx, groupID)
}
func (m *MockService) GetFamilyMembersByIDs(ctx context.Context, ids []primitive.ObjectID) ([]models.FamilyMember, error) {
	return m.GetFamilyMembersByIDsFunc(ctx, ids)
}
func (m *MockService) RemoveGroupIDFromAllFamilyMembers(ctx context.Context, groupID primitive.ObjectID) error {
	return m.RemoveGroupIDFromAllFamilyMembersFunc(ctx, groupID)
}
func (m *MockService) CountGroupsByName(ctx context.Context, name string) (int64, error) {
	return m.CountGroupsByNameFunc(ctx, name)
}
func (m *MockService) CreateGroup(ctx context.Context, group *models.Group) error {
	return m.CreateGroupFunc(ctx, group)
}
func (m *MockService) GetGroup(ctx context.Context, id primitive.ObjectID) (*models.Group, error) {
	return m.GetGroupFunc(ctx, id)
}
func (m *MockService) GetGroupByCode(ctx context.Context, code string) (*models.Group, error) {
	return m.GetGroupByCodeFunc(ctx, code)
}
func (m *MockService) GetGroups(ctx context.Context) ([]models.Group, error) {
	return m.GetGroupsFunc(ctx)
}
func (m *MockService) UpdateGroup(ctx context.Context, id primitive.ObjectID, update bson.M) error {
	return m.UpdateGroupFunc(ctx, id, update)
}
func (m *MockService) DeleteGroup(ctx context.Context, id primitive.ObjectID) error {
	return m.DeleteGroupFunc(ctx, id)
}
func (m *MockService) CreateEvent(ctx context.Context, event *models.Event) error {
	return m.CreateEventFunc(ctx, event)
}
func (m *MockService) GetEvent(ctx context.Context, id primitive.ObjectID) (*models.Event, error) {
	return m.GetEventFunc(ctx, id)
}
func (m *MockService) GetEventByCode(ctx context.Context, code string) (*models.Event, error) {
	return m.GetEventByCodeFunc(ctx, code)
}
func (m *MockService) GetEventsByGroupID(ctx context.Context, groupID primitive.ObjectID, includeCompleted bool) ([]models.Event, error) {
	return m.GetEventsByGroupIDFunc(ctx, groupID, includeCompleted)
}
func (m *MockService) GetEventsByUserID(ctx context.Context, userID primitive.ObjectID) ([]models.Event, error) {
	return m.GetEventsByUserIDFunc(ctx, userID)
}
func (m *MockService) UpdateEvent(ctx context.Context, id primitive.ObjectID, update bson.M) error {
	return m.UpdateEventFunc(ctx, id, update)
}
func (m *MockService) DeleteEvent(ctx context.Context, id primitive.ObjectID) error {
	return m.DeleteEventFunc(ctx, id)
}
func (m *MockService) GetCompletedEventsByRecurrenceID(ctx context.Context, recurrenceID primitive.ObjectID) ([]models.Event, error) {
	return m.GetCompletedEventsByRecurrenceIDFunc(ctx, recurrenceID)
}
func (m *MockService) CreateDish(ctx context.Context, dish *models.Dish) error {
	return m.CreateDishFunc(ctx, dish)
}
func (m *MockService) GetDishesByEventID(ctx context.Context, eventID primitive.ObjectID) ([]models.Dish, error) {
	return m.GetDishesByEventIDFunc(ctx, eventID)
}
func (m *MockService) GetDishByID(ctx context.Context, id primitive.ObjectID) (*models.Dish, error) {
	return m.GetDishByIDFunc(ctx, id)
}
func (m *MockService) UpdateDish(ctx context.Context, id primitive.ObjectID, update bson.M) error {
	return m.UpdateDishFunc(ctx, id, update)
}
func (m *MockService) DeleteDish(ctx context.Context, id primitive.ObjectID) error {
	return m.DeleteDishFunc(ctx, id)
}
func (m *MockService) UpsertRSVP(ctx context.Context, rsvp *models.RSVP) (primitive.ObjectID, error) {
	return m.UpsertRSVPFunc(ctx, rsvp)
}
func (m *MockService) GetRSVPsByEventID(ctx context.Context, eventID primitive.ObjectID) ([]models.RSVP, error) {
	return m.GetRSVPsByEventIDFunc(ctx, eventID)
}
func (m *MockService) CreateSwapRequest(ctx context.Context, swap *models.SwapRequest) error {
	return m.CreateSwapRequestFunc(ctx, swap)
}
func (m *MockService) GetSwapRequestsByEventID(ctx context.Context, eventID primitive.ObjectID) ([]models.SwapRequest, error) {
	return m.GetSwapRequestsByEventIDFunc(ctx, eventID)
}
func (m *MockService) GetSwapRequestByID(ctx context.Context, id primitive.ObjectID) (*models.SwapRequest, error) {
	return m.GetSwapRequestByIDFunc(ctx, id)
}
func (m *MockService) UpdateSwapRequest(ctx context.Context, id primitive.ObjectID, update bson.M) error {
	return m.UpdateSwapRequestFunc(ctx, id, update)
}
func (m *MockService) CreateChatMessage(ctx context.Context, msg *models.ChatMessage) error {
	return m.CreateChatMessageFunc(ctx, msg)
}
func (m *MockService) GetChatMessagesByEventID(ctx context.Context, eventID primitive.ObjectID) ([]models.ChatMessage, error) {
	return m.GetChatMessagesByEventIDFunc(ctx, eventID)
}
func (m *MockService) CreateHousehold(ctx context.Context, household *models.Household) error {
	return m.CreateHouseholdFunc(ctx, household)
}
func (m *MockService) GetHousehold(ctx context.Context, id primitive.ObjectID) (*models.Household, error) {
	return m.GetHouseholdFunc(ctx, id)
}
func (m *MockService) UpdateHousehold(ctx context.Context, id primitive.ObjectID, update bson.M) error {
	return m.UpdateHouseholdFunc(ctx, id, update)
}
func (m *MockService) DeleteHousehold(ctx context.Context, id primitive.ObjectID) error {
	return m.DeleteHouseholdFunc(ctx, id)
}
func (m *MockService) RemoveMemberFromHousehold(ctx context.Context, householdID, familyID primitive.ObjectID) error {
	return m.RemoveMemberFromHouseholdFunc(ctx, householdID, familyID)
}
