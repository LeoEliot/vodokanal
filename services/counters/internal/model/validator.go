package model

import (
	"errors"
	"fmt"
	"regexp"
	"time"
)

var (
	ErrInvalidSerialNumber    = errors.New("invalid serial number format")
	ErrInvalidInstallationDate = errors.New("installation date cannot be in the future")
	ErrInvalidReplacementDate  = errors.New("replacement date cannot be in the future")
	ErrDuplicateSerialNumber   = errors.New("serial number already exists")
)

// CounterValidator validates counter data
type CounterValidator struct {
	SerialNumberPattern *regexp.Regexp
	MaxInstallationYears int
}

// DefaultCounterValidator returns a validator with default rules
func DefaultCounterValidator() *CounterValidator {
	return &CounterValidator{
		SerialNumberPattern: regexp.MustCompile(`^[A-Z0-9\-]{4,30}$`),
		MaxInstallationYears: 50, // Counter cannot be older than 50 years
	}
}

// ValidateCreate validates a create counter request
func (v *CounterValidator) ValidateCreate(req *CreateCounterRequest) error {
	// Validate serial number
	if !v.SerialNumberPattern.MatchString(req.SerialNumber) {
		return ErrInvalidSerialNumber
	}

	// Parse installation date
	installationDate, err := time.Parse("2006-01-02", req.InstallationDate)
	if err != nil {
		return fmt.Errorf("invalid installation date format: %w", err)
	}

	// Check if date is in the future
	if installationDate.After(time.Now()) {
		return ErrInvalidInstallationDate
	}

	// Check if date is too old (more than MaxInstallationYears)
	cutoffDate := time.Now().AddDate(-v.MaxInstallationYears, 0, 0)
	if installationDate.Before(cutoffDate) {
		return fmt.Errorf("installation date is too old (more than %d years)", v.MaxInstallationYears)
	}

	// Validate initial value
	if req.InitialValue != nil && *req.InitialValue < 0 {
		return errors.New("initial value cannot be negative")
	}

	return nil
}

// ValidateUpdate validates an update counter request
func (v *CounterValidator) ValidateUpdate(req *UpdateCounterRequest) error {
	// Validate serial number if provided
	if req.SerialNumber != nil {
		if !v.SerialNumberPattern.MatchString(*req.SerialNumber) {
			return ErrInvalidSerialNumber
		}
	}

	// Validate type if provided
	if req.Type != nil {
		if *req.Type != CounterTypeCold && *req.Type != CounterTypeHot {
			return errors.New("invalid counter type")
		}
	}

	// Validate verification date if provided
	if req.VerificationDate != nil {
		_, err := time.Parse("2006-01-02", *req.VerificationDate)
		if err != nil {
			return fmt.Errorf("invalid verification date format: %w", err)
		}
	}

	return nil
}

// ValidateVerification validates a verification request
func (v *CounterValidator) ValidateVerification(req *VerificationRequest) error {
	// Parse verification date
	verificationDate, err := time.Parse("2006-01-02", req.VerificationDate)
	if err != nil {
		return fmt.Errorf("invalid verification date format: %w", err)
	}

	// Check if date is in the future
	if verificationDate.After(time.Now()) {
		return errors.New("verification date cannot be in the future")
	}

	// Validate verified value
	if req.VerifiedValue != nil && *req.VerifiedValue < 0 {
		return errors.New("verified value cannot be negative")
	}

	return nil
}

// ValidateReplacement validates a replacement request
func (v *CounterValidator) ValidateReplacement(req *ReplacementRequest) error {
	// Validate old counter ID
	if req.OldCounterID <= 0 {
		return errors.New("invalid old counter ID")
	}

	// Validate new serial number
	if !v.SerialNumberPattern.MatchString(req.NewSerialNumber) {
		return ErrInvalidSerialNumber
	}

	// Parse replacement date
	replacementDate, err := time.Parse("2006-01-02", req.ReplacementDate)
	if err != nil {
		return fmt.Errorf("invalid replacement date format: %w", err)
	}

	// Check if date is in the future
	if replacementDate.After(time.Now()) {
		return ErrInvalidReplacementDate
	}

	// Validate values
	if req.FinalValue < 0 {
		return errors.New("final value cannot be negative")
	}

	if req.InitialValue < 0 {
		return errors.New("initial value cannot be negative")
	}

	// Validate type if provided
	if req.NewType != nil {
		if *req.NewType != CounterTypeCold && *req.NewType != CounterTypeHot {
			return errors.New("invalid counter type")
		}
	}

	return nil
}

// CalculateNextVerification calculates when the next verification is due
func CalculateNextVerification(installationDate time.Time, verificationDays int) time.Time {
	// Standard verification period is 4 years (1460 days)
	// But can be configured via VERIFICATION_DAYS
	nextVerification := installationDate.AddDate(4, 0, 0)

	// If there's a custom verification period, use it
	if verificationDays > 0 {
		nextVerification = installationDate.AddDate(0, 0, verificationDays)
	}

	return nextVerification
}

// DaysUntilVerification returns days until verification is due
func DaysUntilVerification(dueDate time.Time) *int {
	if dueDate.IsZero() {
		return nil
	}

	days := int(time.Until(dueDate).Hours() / 24)
	return &days
}
