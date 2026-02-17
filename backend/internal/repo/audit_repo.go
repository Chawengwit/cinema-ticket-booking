package repo

import (
	"cinema/internal/model"
	"context"

	"go.mongodb.org/mongo-driver/mongo"
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
