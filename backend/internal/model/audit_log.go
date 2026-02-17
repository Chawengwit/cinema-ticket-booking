package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AuditLog struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Type       string             `bson:"type" json:"type"` // seat.locked, seat.released, seat.booked, seat.timeout, booking.success
	ShowtimeID string             `bson:"showtime_id,omitempty" json:"showtime_id,omitempty"`
	BookingID  string             `bson:"booking_id,omitempty" json:"booking_id,omitempty"`
	UserID     string             `bson:"user_id,omitempty" json:"user_id,omitempty"`
	SeatIDs    []string           `bson:"seat_ids,omitempty" json:"seat_ids,omitempty"`
	RequestID  string             `bson:"request_id,omitempty" json:"request_id,omitempty"`
	Payload    any                `bson:"payload,omitempty" json:"payload,omitempty"` // เก็บ raw event
	At         time.Time          `bson:"at" json:"at"`
}
