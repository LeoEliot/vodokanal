package model

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrInvalidValue        = errors.New("reading value must be positive")
	ErrInvalidDate        = errors.New("reading date cannot be in the future")
	ErrInvalidDateRange   = errors.New("reading date is too old")
	ErrValueTooLow        = errors.New("reading value is lower than previous")
	ErrValueTooHigh       = errors.New("reading value increase is unrealistic")
)

// ReadingValidator validates reading values
type ReadingValidator struct {
	MaxDaysBack       int     // Maximum days back for reading date
	MaxIncreasePercent float64 // Maximum allowed value increase percent
	AllowDecrease     bool    // Allow value decrease (for counter replacements)
}

// DefaultReadingValidator returns a validator with default rules
func DefaultReadingValidator() *ReadingValidator {
	return &ReadingValidator{
		MaxDaysBack:       90, // 3 months
		MaxIncreasePercent: 50, // 50% increase
		AllowDecrease:     true,
	}
}

// ValidateCreate validates a create reading request
func (v *ReadingValidator) ValidateCreate(req *CreateReadingRequest, previousValue float64) error {
	// Validate value
	if req.Value <= 0 {
		return ErrInvalidValue
	}

	// Parse and validate date
	readingDate, err := time.Parse("2006-01-02", req.ReadingDate)
	if err != nil {
		// If date is empty, use today
		if req.ReadingDate == "" {
			return nil
		}
		return fmt.Errorf("invalid date format: %w", err)
	}

	// Check if date is in the future
	if readingDate.After(time.Now()) {
		return ErrInvalidDate
	}

	// Check if date is too old
	cutoffDate := time.Now().AddDate(0, 0, -v.MaxDaysBack)
	if readingDate.Before(cutoffDate) {
		return ErrInvalidDateRange
	}

	// Validate against previous value
	if previousValue > 0 {
		// Check for unrealistic increase
		increasePercent := ((req.Value - previousValue) / previousValue) * 100
		if increasePercent > v.MaxIncreasePercent {
			return fmt.Errorf("%w: increase of %.1f%% exceeds maximum %.0f%%",
				ErrValueTooHigh, increasePercent, v.MaxIncreasePercent)
		}

		// Check for decrease (only if not allowed)
		if !v.AllowDecrease && req.Value < previousValue {
			return fmt.Errorf("%w: current %.2f, previous %.2f",
				ErrValueTooLow, req.Value, previousValue)
		}
	}

	return nil
}

// ValidateUpdate validates an update reading request
func (v *ReadingValidator) ValidateUpdate(req *UpdateReadingRequest) error {
	if req.Value != nil && *req.Value <= 0 {
		return ErrInvalidValue
	}

	if req.Verified != nil && *req.Verified && req.VerifiedBy == nil {
		return errors.New("verified_by is required when verifying")
	}

	return nil
}

// ParseDate parses a date string or returns current time if empty
func ParseDate(dateStr string) (time.Time, error) {
	if dateStr == "" {
		return time.Now(), nil
	}
	return time.Parse("2006-01-02", dateStr)
}
