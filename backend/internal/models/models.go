package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type FamilyMember struct {
	ID                 primitive.ObjectID   `json:"id" bson:"_id,omitempty"`
	Name               string               `json:"name" bson:"name"`
	Email              string               `json:"email" bson:"email"`
	GoogleID           string               `json:"google_id" bson:"google_id"`
	Picture            string               `json:"picture" bson:"picture"`
	Address            string               `json:"address" bson:"address"`
	Allergies          string               `json:"allergies" bson:"allergies"`
	DietaryPreferences []string             `json:"dietary_preferences" bson:"dietary_preferences"` // e.g., ["Vegan", "Gluten-Free"]
	GroupIDs           []primitive.ObjectID `json:"group_ids" bson:"group_ids,omitempty"`
	HouseholdID        *primitive.ObjectID  `json:"household_id,omitempty" bson:"household_id,omitempty"`
}

// SafeFamilyMember is a version of FamilyMember with sensitive fields omitted for API responses
type SafeFamilyMember struct {
	ID                 primitive.ObjectID   `json:"id"`
	Name               string               `json:"name"`
	Email              string               `json:"email"`
	Picture            string               `json:"picture"`
	DietaryPreferences []string             `json:"dietary_preferences"`
	GroupIDs           []primitive.ObjectID `json:"group_ids"`
	HouseholdID        *primitive.ObjectID  `json:"household_id,omitempty"`
}

func (f *FamilyMember) ToSafe() SafeFamilyMember {
	return SafeFamilyMember{
		ID:                 f.ID,
		Name:               f.Name,
		Email:              f.Email,
		Picture:            f.Picture,
		DietaryPreferences: f.DietaryPreferences,
		GroupIDs:           f.GroupIDs,
		HouseholdID:        f.HouseholdID,
	}
}

type Household struct {
	ID        primitive.ObjectID   `json:"id" bson:"_id,omitempty"`
	Name      string               `json:"name" bson:"name"` // e.g., "The Smiths"
	Address   string               `json:"address" bson:"address"`
	MemberIDs []primitive.ObjectID `json:"member_ids" bson:"member_ids"` // IDs of Family members
	Members   []SafeFamilyMember   `json:"members,omitempty" bson:"-"`   // Full member details for response
}

type Group struct {
	ID       primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name     string             `json:"name" bson:"name"`
	AdminID  primitive.ObjectID `json:"admin_id" bson:"admin_id"`
	JoinCode string             `json:"join_code" bson:"join_code"`
}

type Event struct {
	ID              primitive.ObjectID   `json:"id" bson:"_id,omitempty"`
	GroupID         primitive.ObjectID   `json:"group_id" bson:"group_id"`
	Name            string               `json:"name" bson:"name"`
	Date            time.Time            `json:"date" bson:"date"`
	Type            string               `json:"type" bson:"type"` // Dinner, Lunch, Coffee
	HostID          primitive.ObjectID   `json:"host_id" bson:"host_id"`
	HostName        string               `json:"host_name,omitempty" bson:"-"`
	HostHouseholdID *primitive.ObjectID  `json:"host_household_id,omitempty" bson:"-"`
	Location        string               `json:"location" bson:"location"`
	Description     string               `json:"description" bson:"description"`
	Recurrence      string               `json:"recurrence,omitempty" bson:"recurrence,omitempty"`       // Weekly, Bi-Weekly
	RecurrenceID    primitive.ObjectID   `json:"recurrence_id,omitempty" bson:"recurrence_id,omitempty"` // ID linking the series
	GuestIDs        []primitive.ObjectID `json:"guest_ids,omitempty" bson:"guest_ids,omitempty"`
	GuestJoinCode   string               `json:"guest_join_code" bson:"guest_join_code"`
	Status          string               `json:"status" bson:"status"` // scheduled, completed, cancelled
}

type RSVP struct {
	ID                 primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	EventID            primitive.ObjectID `json:"event_id" bson:"event_id"`
	FamilyMemberID     primitive.ObjectID `json:"family_id" bson:"family_id"`
	FamilyName         string             `json:"family_name,omitempty" bson:"-"`
	FamilyPicture      string             `json:"family_picture,omitempty" bson:"-"`
	Status             string             `json:"status" bson:"status"` // Yes, No, Maybe
	Count              int                `json:"count" bson:"count"`   // Total count or Adult count
	KidsCount          int                `json:"kids_count" bson:"kids_count"`
	DietaryPreferences []string           `json:"dietary_preferences,omitempty" bson:"-"`
}

type Dish struct {
	ID          primitive.ObjectID  `json:"id" bson:"_id,omitempty"`
	EventID     primitive.ObjectID  `json:"event_id" bson:"event_id"`
	Name        string              `json:"name" bson:"name"`
	Description string              `json:"description" bson:"description"`
	DietaryTags []string            `json:"dietary_tags" bson:"dietary_tags"`       // e.g., ["Vegan", "Gluten-Free"]
	BringerID   *primitive.ObjectID `json:"bringer_id" bson:"bringer_id,omitempty"` // Nullable
	BringerName string              `json:"bringer_name,omitempty" bson:"-"`
	IsHostDish  bool                `json:"is_host_dish" bson:"is_host_dish"`
	IsRequested bool                `json:"is_requested" bson:"is_requested"`
	IsSuggested bool                `json:"is_suggested" bson:"is_suggested"`
}

type SwapRequest struct {
	ID                       primitive.ObjectID  `json:"id" bson:"_id,omitempty"`
	EventID                  primitive.ObjectID  `json:"event_id" bson:"event_id"`
	DishID                   *primitive.ObjectID `json:"dish_id,omitempty" bson:"dish_id,omitempty"`
	Type                     string              `json:"type" bson:"type"` // "dish" or "host"
	RequestingFamilyMemberID primitive.ObjectID  `json:"requesting_family_id" bson:"requesting_family_id"`
	RequestingFamilyName     string              `json:"requesting_family_name,omitempty" bson:"-"`
	TargetFamilyMemberID     *primitive.ObjectID `json:"target_family_id" bson:"target_family_id,omitempty"`
	TargetFamilyName         string              `json:"target_family_name,omitempty" bson:"-"`
	Status                   string              `json:"status" bson:"status"` // Pending, Approved, Rejected
	CreatedAt                time.Time           `json:"created_at" bson:"created_at"`
}

type ChatMessage struct {
	ID             primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	EventID        primitive.ObjectID `json:"event_id" bson:"event_id"`
	FamilyMemberID primitive.ObjectID `json:"family_id" bson:"family_id"`
	FamilyName     string             `json:"family_name" bson:"family_name"`
	Content        string             `json:"content" bson:"content"`
	CreatedAt      time.Time          `json:"created_at" bson:"created_at"`
}
