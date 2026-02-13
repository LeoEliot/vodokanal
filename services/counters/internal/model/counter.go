package model

import "time"

// CounterType represents the type of water meter
type CounterType string

const (
	CounterTypeCold CounterType = "cold"
	CounterTypeHot  CounterType = "hot"
)

// Counter represents a water meter
type Counter struct {
	ID              int        `json:"id" db:"id"`
	SubscriberID    int        `json:"subscriber_id" db:"subscriber_id"`
	SerialNumber     string     `json:"serial_number" db:"serial_number"`
	Type            CounterType `json:"type" db:"type"`
	InstallationDate  time.Time  `json:"installation_date" db:"installation_date"`
	VerificationDate  *time.Time `json:"verification_date,omitempty" db:"verification_date"`
	NextVerification *time.Time `json:"next_verification,omitempty" db:"-"`
	IsActive        bool       `json:"is_active" db:"is_active"`
	InitialValue    *float64   `json:"initial_value,omitempty" db:"initial_value"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
}

// CounterWithDetails represents a counter with subscriber information
type CounterWithDetails struct {
	ID              int        `json:"id"`
	SubscriberID    int        `json:"subscriber_id"`
	SubscriberName  string     `json:"subscriber_name,omitempty"`
	AccountNumber   string     `json:"account_number,omitempty"`
	SerialNumber     string     `json:"serial_number"`
	Type            CounterType `json:"type"`
	InstallationDate  time.Time  `json:"installation_date"`
	VerificationDate  *time.Time `json:"verification_date,omitempty"`
	NextVerification *time.Time `json:"next_verification,omitempty"`
	IsActive        bool       `json:"is_active"`
	InitialValue    *float64   `json:"initial_value,omitempty"`
	CurrentValue    *float64   `json:"current_value,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// CounterWithReadings represents a counter with its readings
type CounterWithReadings struct {
	Counter
	LatestReading   *float64   `json:"latest_reading,omitempty"`
	ReadingDate     *time.Time `json:"reading_date,omitempty"`
	TotalReadings   int        `json:"total_readings,omitempty"`
}

// CreateCounterRequest represents a request to create a counter
type CreateCounterRequest struct {
	SubscriberID   int        `json:"subscriber_id" binding:"required"`
	SerialNumber    string     `json:"serial_number" binding:"required"`
	Type           CounterType `json:"type" binding:"required,oneof=cold hot"`
	InstallationDate string     `json:"installation_date" binding:"required"`
	InitialValue   *float64   `json:"initial_value"`
}

// UpdateCounterRequest represents a request to update a counter
type UpdateCounterRequest struct {
	SerialNumber    *string     `json:"serial_number"`
	Type           *CounterType `json:"type" binding:"omitempty,oneof=cold hot"`
	VerificationDate *string     `json:"verification_date"`
	IsActive       *bool       `json:"is_active"`
}

// VerificationRequest represents a verification request
type VerificationRequest struct {
	VerificationDate string  `json:"verification_date" binding:"required"`
	VerifiedValue   *float64 `json:"verified_value"`
	Notes           string  `json:"notes"`
}

// ReplacementRequest represents a counter replacement request
type ReplacementRequest struct {
	OldCounterID      int     `json:"old_counter_id" binding:"required"`
	NewSerialNumber   string  `json:"new_serial_number" binding:"required"`
	ReplacementDate   string  `json:"replacement_date" binding:"required"`
	NewType          *CounterType `json:"new_type"`
	FinalValue        float64 `json:"final_value" binding:"required"`
	InitialValue      float64 `json:"initial_value" binding:"required"`
	Reason            string  `json:"reason"`
}

// ListResult represents paginated counter list result
type ListResult struct {
	Data       []CounterWithDetails `json:"data"`
	Page       int                `json:"page"`
	Limit      int                `json:"limit"`
	Total      int                `json:"total"`
	TotalPages int                `json:"total_pages"`
}

// CounterStatus represents the status of a counter
type CounterStatus struct {
	CounterID         int     `json:"counter_id"`
	SerialNumber      string  `json:"serial_number"`
	Type             string  `json:"type"`
	IsActive         bool     `json:"is_active"`
	NeedsVerification bool     `json:"needs_verification"`
	DaysUntilVerification *int  `json:"days_until_verification,omitempty"`
	LastReading      *float64 `json:"last_reading,omitempty"`
	LastReadingDate  *string  `json:"last_reading_date,omitempty"`
}

// Dashboard represents counter dashboard data
type Dashboard struct {
	TotalCounters    int                    `json:"total_counters"`
	ActiveCounters   int                    `json:"active_counters"`
	ColdCounters     int                    `json:"cold_counters"`
	HotCounters      int                    `json:"hot_counters"`
	NeedsVerification int                    `json:"needs_verification"`
	ByType           map[string]int          `json:"by_type"`
	ByStatus         map[string]int          `json:"by_status"`
	Recent           []CounterWithDetails   `json:"recent,omitempty"`
}
