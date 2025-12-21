package database

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (s *service) DeleteHousehold(ctx context.Context, id primitive.ObjectID) error {
	// First, find all members and remove their household_id
	_, err := s.db.Collection("families").UpdateMany(
		ctx,
		bson.M{"household_id": id},
		bson.M{"$unset": bson.M{"household_id": ""}},
	)
	if err != nil {
		return err
	}

	// Then delete the household
	_, err = s.db.Collection("households").DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (s *service) RemoveMemberFromHousehold(ctx context.Context, householdID, familyMemberID primitive.ObjectID) error {
	// Remove member from household's member_ids
	_, err := s.db.Collection("households").UpdateOne(
		ctx,
		bson.M{"_id": householdID},
		bson.M{"$pull": bson.M{"member_ids": familyMemberID}},
	)
	if err != nil {
		return err
	}

	// Remove household_id from family
	_, err = s.db.Collection("families").UpdateOne(
		ctx,
		bson.M{"_id": familyMemberID},
		bson.M{"$unset": bson.M{"household_id": ""}},
	)
	return err
}
