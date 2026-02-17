package repo

import (
	"cinema/internal/model"
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type BookingRepo struct {
	col *mongo.Collection
}

func NewBookingRepo(db *mongo.Database) *BookingRepo {
	return &BookingRepo{col: db.Collection("bookings")}
}

// CreatePending creates a booking document with status=PENDING.
func (r *BookingRepo) CreatePending(ctx context.Context, b *model.Booking) error {
	if b == nil {
		return mongo.ErrNilDocument
	}

	now := time.Now()
	b.Status = model.BookingPending
	b.CreatedAt = now
	b.UpdatedAt = now

	_, err := r.col.InsertOne(ctx, b)
	return err
}

func (r *BookingRepo) MarkBooked(ctx context.Context, bookingID primitive.ObjectID, paymentRef string) error {
	now := time.Now()
	_, err := r.col.UpdateByID(ctx, bookingID, bson.M{
		"$set": bson.M{
			"status":      model.BookingBooked,
			"payment_ref": paymentRef,
			"booked_at":   now,
			"updated_at":  now,
		},
	})
	return err
}

func (r *BookingRepo) MarkFailed(ctx context.Context, bookingID primitive.ObjectID) error {
	now := time.Now()
	_, err := r.col.UpdateByID(ctx, bookingID, bson.M{
		"$set": bson.M{
			"status":     model.BookingFailed,
			"updated_at": now,
		},
	})
	return err
}
