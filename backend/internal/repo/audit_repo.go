package repo

import (
	"cinema/internal/model"
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AuditRepo struct {
	col *mongo.Collection
}

func NewAuditRepo(db *mongo.Database) *AuditRepo {
	return &AuditRepo{col: db.Collection("audit_logs")}
}

func (r *AuditRepo) Insert(ctx context.Context, a *model.AuditLog) error {
	if a == nil {
		return mongo.ErrNilDocument
	}
	_, err := r.col.InsertOne(ctx, a)
	return err
}

// ===== Admin query =====
type AdminAuditFilter struct {
	Type       string
	ShowtimeID string
	UserID     string
	BookingID  string

	From *time.Time // at >= from
	To   *time.Time // at <= to

	Limit int64
	Skip  int64
}

func (r *AuditRepo) FindAdmin(ctx context.Context, f AdminAuditFilter) ([]model.AuditLog, int64, error) {
	q := bson.M{}

	if f.Type != "" {
		q["type"] = f.Type
	}
	if f.ShowtimeID != "" {
		q["showtime_id"] = f.ShowtimeID
	}
	if f.UserID != "" {
		q["user_id"] = f.UserID
	}
	if f.BookingID != "" {
		q["booking_id"] = f.BookingID
	}
	if f.From != nil || f.To != nil {
		rng := bson.M{}
		if f.From != nil {
			rng["$gte"] = *f.From
		}
		if f.To != nil {
			rng["$lte"] = *f.To
		}
		q["at"] = rng
	}

	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
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
		SetSort(bson.D{{Key: "at", Value: -1}}).
		SetLimit(limit).
		SetSkip(skip)

	cur, err := r.col.Find(ctx, q, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)

	out := make([]model.AuditLog, 0)
	if err := cur.All(ctx, &out); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}
