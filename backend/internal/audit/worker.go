package audit

import (
	"cinema/internal/model"
	"cinema/internal/repo"
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type seatEvent struct {
	Type       string   `json:"type"`
	ShowtimeID string   `json:"showtime_id"`
	SeatIDs    []string `json:"seat_ids"`
	Owner      string   `json:"owner"`
	RequestID  string   `json:"request_id,omitempty"`
	BookingID  string   `json:"booking_id,omitempty"`
	At         int64    `json:"at"`
}

type bookingEvent struct {
	Type      string   `json:"type"`
	BookingID string   `json:"booking_id"`
	Showtime  string   `json:"showtime_id"`
	UserID    string   `json:"user_id"`
	SeatIDs   []string `json:"seat_ids"`
	Amount    int64    `json:"amount"`
	Currency  string   `json:"currency"`
	At        int64    `json:"at"`
}

func Run(ctx context.Context, rdb *redis.Client, audits *repo.AuditRepo) {
	// pattern สำหรับ seat-events:*
	ps := rdb.PSubscribe(ctx, "seat-events:*")
	// channel booking-events
	bs := rdb.Subscribe(ctx, "booking-events")

	defer func() { _ = ps.Close() }()
	defer func() { _ = bs.Close() }()

	seatCh := ps.Channel()
	bookCh := bs.Channel()

	for {
		select {
		case <-ctx.Done():
			return

		case msg := <-seatCh:
			var ev seatEvent
			if err := json.Unmarshal([]byte(msg.Payload), &ev); err != nil {
				continue
			}

			// map เป็น audit type ให้ชัด
			t := "seat." + strings.ToLower(ev.Type) // locked/released/booked/timeout
			at := time.Unix(ev.At, 0)
			if ev.At == 0 {
				at = time.Now()
			}

			_ = audits.Insert(ctx, &model.AuditLog{
				Type:       t,
				ShowtimeID: ev.ShowtimeID,
				BookingID:  ev.BookingID,
				UserID:     ev.Owner,
				SeatIDs:    ev.SeatIDs,
				RequestID:  ev.RequestID,
				Payload:    json.RawMessage([]byte(msg.Payload)),
				At:         at,
			})

		case msg := <-bookCh:
			var ev bookingEvent
			if err := json.Unmarshal([]byte(msg.Payload), &ev); err != nil {
				continue
			}

			at := time.Unix(ev.At, 0)
			if ev.At == 0 {
				at = time.Now()
			}

			if err := audits.Insert(ctx, &model.AuditLog{
				Type:       ev.Type, // booking.success
				ShowtimeID: ev.Showtime,
				BookingID:  ev.BookingID,
				UserID:     ev.UserID,
				SeatIDs:    ev.SeatIDs,
				Payload:    json.RawMessage([]byte(msg.Payload)),
				At:         at,
			}); err != nil {
				log.Println("audit insert failed:", err)
			}
		}
	}
}
