package model

import "time"

// Reading represents a meter reading
type Reading struct {
	ID           int       `json:"id" db:"id"`
	SubscriberID int       `json:"subscriber_id" db:"subscriber_id"`
	CounterID    int       `json:"counter_id" db:"counter_id"`
	Value        float64   `json:"value" db:"value"`
	ReadingDate  time.Time `json:"reading_date" db:"reading_date"`
	SubmittedBy  *int      `json:"submitted_by,omitempty" db:"submitted_by"`
	Verified     bool      `json:"verified" db:"verified"`
	VerifiedBy   *int      `json:"verified_by,omitempty" db:"verified_by"`
	VerifiedAt   *time.Time `json:"verified_at,omitempty" db:"verified_at"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// CreateReadingRequest represents a request to create a reading
type CreateReadingRequest struct {
	SubscriberID int     `json:"subscriber_id" binding:"required"`
	CounterID    int     `json:"counter_id" binding:"required"`
	Value        float64 `json:"value" binding:"required,gt=0"`
	ReadingDate  string  `json:"reading_date"` // Format: YYYY-MM-DD
}

// UpdateReadingRequest represents a request to update a reading
type UpdateReadingRequest struct {
	Value      *float64 `json:"value"`
	Verified   *bool    `json:"verified"`
	VerifiedBy *int      `json:"verified_by"`
}

// ReadingWithDetails represents a reading with extended information
type ReadingWithDetails struct {
	ID           int       `json:"id"`
	SubscriberID int       `json:"subscriber_id"`
	SubscriberName string  `json:"subscriber_name,omitempty"`
	CounterID    int       `json:"counter_id"`
	CounterSerial string   `json:"counter_serial,omitempty"`
	CounterType  string   `json:"counter_type,omitempty"`
	Value        float64   `json:"value"`
	ReadingDate  time.Time `json:"reading_date"`
	SubmittedBy  *int      `json:"submitted_by,omitempty"`
	Verified     bool      `json:"verified"`
	VerifiedBy   *int      `json:"verified_by,omitempty"`
	VerifiedAt   *time.Time `json:"verified_at,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// ListResult represents paginated reading list result
type ListResult struct {
	Data       []ReadingWithDetails `json:"data"`
	Page       int                `json:"page"`
	Limit      int                `json:"limit"`
	Total      int                `json:"total"`
	TotalPages int                `json:"total_pages"`
}

// Statistics represents reading statistics
type Statistics struct {
	SubscriberID   int     `json:"subscriber_id"`
	CounterID      int     `json:"counter_id"`
	CurrentValue  float64 `json:"current_value"`
	PreviousValue float64 `json:"previous_value"`
	Consumption   float64 `json:"consumption"`
	Period        string  `json:"period"`
}

// MonthlyReadings represents readings grouped by month
type MonthlyReadings struct {
	Month   string    `json:"month"` // YYYY-MM
	Readings []Reading `json:"readings"`
	Total    float64   `json:"total"`
}
