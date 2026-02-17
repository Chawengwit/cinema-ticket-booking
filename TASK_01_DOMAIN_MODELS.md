# Task 01: Domain Models & Repositories

**Status**: Ready for Implementation  
**Owner**: Backend Team  
**Context**: Before building REST APIs, we must define the core data structures that map to MongoDB and the Go application logic.

## 1. Objective
Create the `internal/domain` and `internal/repository` packages. Define Go structs with BSON tags for persistence and JSON tags for API serialization. Implement the basic CRUD interfaces for MongoDB.

## 2. Domain Entities (`internal/domain`)

### A. Movie
Represents the film metadata.
```go
type Movie struct {
    ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    Title       string             `bson:"title" json:"title"`
    Description string             `bson:"description" json:"description"`
    DurationMin int                `bson:"duration_min" json:"duration_min"` // e.g., 120
    Genre       []string           `bson:"genre" json:"genre"`
    PosterURL   string             `bson:"poster_url" json:"poster_url"`
    CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
}
```

### B. Showtime
A specific screening of a movie. Contains the seat layout snapshot to ensure historical accuracy if the venue changes later.
```go
type Showtime struct {
    ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    MovieID   primitive.ObjectID `bson:"movie_id" json:"movie_id"`
    StartTime time.Time          `bson:"start_time" json:"start_time"`
    AuditoriumID string          `bson:"auditorium_id" json:"auditorium_id"`
    // Snapshot of the layout (Rows x Cols) for rendering the seat map
    Layout    SeatLayout         `bson:"layout" json:"layout"` 
}

type SeatLayout struct {
    Rows int `bson:"rows" json:"rows"`
    Cols int `bson:"cols" json:"cols"`
    // Unavailable seats (e.g., gaps in the theater)
    Unavailable []string `bson:"unavailable" json:"unavailable"` 
}
```

### C. Booking
The transactional record.
```go
type BookingStatus string

const (
    BookingPending   BookingStatus = "PENDING"   // Held by Redis lock, not paid
    BookingConfirmed BookingStatus = "CONFIRMED" // Paid and persisted
    BookingCancelled BookingStatus = "CANCELLED" // Timeout or user cancel
)

type Booking struct {
    ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    UserID     string             `bson:"user_id" json:"user_id"` // From Google Sub
    ShowtimeID primitive.ObjectID `bson:"showtime_id" json:"showtime_id"`
    SeatIDs    []string           `bson:"seat_ids" json:"seat_ids"` // e.g., "A1", "A2"
    Status     BookingStatus      `bson:"status" json:"status"`
    Amount     float64            `bson:"amount" json:"amount"`
    CreatedAt  time.Time          `bson:"created_at" json:"created_at"`
    ExpiresAt  time.Time          `bson:"expires_at" json:"expires_at"` // For pending cleanup
}
```

## 3. Repository Interfaces (`internal/domain`)
Define interfaces here to allow mocking in tests later.

```go
type MovieRepository interface {
    Create(ctx context.Context, movie *Movie) error
    GetByID(ctx context.Context, id string) (*Movie, error)
    List(ctx context.Context) ([]*Movie, error)
}

type ShowtimeRepository interface {
    Create(ctx context.Context, showtime *Showtime) error
    GetByID(ctx context.Context, id string) (*Showtime, error)
    ListByMovie(ctx context.Context, movieID string) ([]*Showtime, error)
}

type BookingRepository interface {
    Create(ctx context.Context, booking *Booking) error
    UpdateStatus(ctx context.Context, id string, status BookingStatus) error
    GetByUser(ctx context.Context, userID string) ([]*Booking, error)
}
```

## 4. Implementation Plan (`internal/repository`)
1. Create `internal/domain/models.go` with the structs above.
2. Create `internal/repository/mongo_movie.go` implementing `MovieRepository`.
3. Create `internal/repository/mongo_showtime.go` implementing `ShowtimeRepository`.
4. Create `internal/repository/mongo_booking.go` implementing `BookingRepository`.
5. Wire these into `cmd/api/main.go` (temporarily log success on boot).