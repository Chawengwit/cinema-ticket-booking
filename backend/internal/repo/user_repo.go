package repo

import (
	"cinema/internal/model"
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type UserRepo struct {
	col *mongo.Collection
}

func NewUserRepo(db *mongo.Database) *UserRepo {
	return &UserRepo{
		col: db.Collection("users"),
	}
}

// Upsert by google_id
func (r *UserRepo) UpsertGoogleUser(ctx context.Context, googleID, email, name, picture string) (*model.User, error) {
	now := time.Now()

	update := bson.M{
		"$set": bson.M{
			"email":      email,
			"name":       name,
			"picture":    picture,
			"updated_at": now,
		},
		"$setOnInsert": bson.M{
			"google_id":  googleID,
			"role":       model.RoleUser, // default
			"created_at": now,
		},
	}

	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)

	var out model.User
	if err := r.col.FindOneAndUpdate(ctx, bson.M{"google_id": googleID}, update, opts).Decode(&out); err != nil {
		return nil, err
	}

	return &out, nil
}

// NEW: update role by Mongo _id
func (r *UserRepo) SetRoleByID(ctx context.Context, id primitive.ObjectID, role model.UserRole) (*model.User, error) {
	now := time.Now()

	update := bson.M{
		"$set": bson.M{
			"role":       role,
			"updated_at": now,
		},
	}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var out model.User
	if err := r.col.FindOneAndUpdate(ctx, bson.M{"_id": id}, update, opts).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

// find user by Mongo _id (hex)
func (r *UserRepo) FindByID(ctx context.Context, idHex string) (*model.User, error) {
	oid, err := primitive.ObjectIDFromHex(idHex)
	if err != nil {
		return nil, err
	}

	var out model.User
	if err := r.col.FindOne(ctx, bson.M{"_id": oid}).Decode(&out); err != nil {
		return nil, err
	}

	return &out, nil
}
