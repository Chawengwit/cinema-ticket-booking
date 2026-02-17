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

// ===== Admin query =====
type AdminBookingFilter struct {
	ShowtimeID string
	Status     model.BookingStatus
	UserID     primitive.ObjectID
	HasUserID  bool

	From *time.Time
	To   *time.Time

	Limit int64
	Skip  int64
}

func (r *BookingRepo) FindAdmin(ctx context.Context, f AdminBookingFilter) ([]model.Booking, int64, error) {
	q := bson.M{}

	if f.ShowtimeID != "" {
		q["showtime_id"] = f.ShowtimeID
	}
	if f.Status != "" {
		q["status"] = f.Status
	}
	if f.HasUserID {
		q["user_id"] = f.UserID
	}
	if f.From != nil || f.To != nil {
		rng := bson.M{}
		if f.From != nil {
			rng["$gte"] = *f.From
		}
		if f.To != nil {
			rng["$lte"] = *f.To
		}
		q["created_at"] = rng
	}

	limit := f.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	skip := f.Skip
	if skip < 0 {
		skip = 0
	}

	total, err := r.col.CountDocuments(ctx, q)
	if err != nil {
		return nil, 0, err
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(limit).
		SetSkip(skip)

	cur, err := r.col.Find(ctx, q, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)

	out := make([]model.Booking, 0)
	if err := cur.All(ctx, &out); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}
