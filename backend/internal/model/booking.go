package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BookingStatus string

const (
	BookingPending BookingStatus = "PENDING"
	BookingBooked  BookingStatus = "BOOKED"
	BookingFailed  BookingStatus = "FAILED"
)

// Booking is created when the user confirms (mock) payment.
type Booking struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ShowtimeID string             `bson:"showtime_id" json:"showtime_id"`
	UserID     primitive.ObjectID `bson:"user_id" json:"user_id"`
	SeatIDs    []string           `bson:"seat_ids" json:"seat_ids"`
	Amount     int64              `bson:"amount" json:"amount"`
	Currency   string             `bson:"currency" json:"currency"`
	Status     BookingStatus      `bson:"status" json:"status"`
	RequestID  string             `bson:"request_id" json:"request_id"`
	PaymentRef string             `bson:"payment_ref,omitempty" json:"payment_ref,omitempty"`
	BookedAt   *time.Time         `bson:"booked_at,omitempty" json:"booked_at,omitempty"`
	CreatedAt  time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt  time.Time          `bson:"updated_at" json:"updated_at"`
}
