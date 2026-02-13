package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestValidatorValidateCreate(t *testing.T) {
	v := DefaultReadingValidator()

	tests := []struct {
		name           string
		req            *CreateReadingRequest
		previousValue  float64
		expectError    bool
		errorContains  string
	}{
		{
			name: "valid reading",
			req: &CreateReadingRequest{
				SubscriberID: 1,
				CounterID:    1,
				Value:        100.0,
				ReadingDate:  time.Now().Format("2006-01-02"),
			},
			previousValue: 90.0,
			expectError:   false,
		},
		{
			name: "zero value",
			req: &CreateReadingRequest{
				SubscriberID: 1,
				CounterID:    1,
				Value:        0,
			},
			previousValue: 0,
			expectError:   true,
			errorContains: "positive",
		},
		{
			name: "negative value",
			req: &CreateReadingRequest{
				SubscriberID: 1,
				CounterID:    1,
				Value:        -10,
			},
			previousValue: 0,
			expectError:   true,
			errorContains: "positive",
		},
		{
			name: "future date",
			req: &CreateReadingRequest{
				SubscriberID: 1,
				CounterID:    1,
				Value:        100.0,
				ReadingDate:  time.Now().AddDate(0, 0, 1).Format("2006-01-02"),
			},
			previousValue: 0,
			expectError:   true,
			errorContains: "future",
		},
		{
			name: "too old date",
			req: &CreateReadingRequest{
				SubscriberID: 1,
				CounterID:    1,
				Value:        100.0,
				ReadingDate:  time.Now().AddDate(0, 0, -100).Format("2006-01-02"),
			},
			previousValue: 0,
			expectError:   true,
			errorContains: "too old",
		},
		{
			name: "unrealistic increase",
			req: &CreateReadingRequest{
				SubscriberID: 1,
				CounterID:    1,
				Value:        200.0,
				ReadingDate:  time.Now().Format("2006-01-02"),
			},
			previousValue: 100.0,
			expectError:   true,
			errorContains: "increase",
		},
		{
			name: "valid increase",
			req: &CreateReadingRequest{
				SubscriberID: 1,
				CounterID:    1,
				Value:        145.0,
				ReadingDate:  time.Now().Format("2006-01-02"),
			},
			previousValue: 100.0,
			expectError:   false,
		},
		{
			name: "decrease allowed",
			req: &CreateReadingRequest{
				SubscriberID: 1,
				CounterID:    1,
				Value:        50.0,
				ReadingDate:  time.Now().Format("2006-01-02"),
			},
			previousValue: 100.0,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateCreate(tt.req, tt.previousValue)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidatorValidateUpdate(t *testing.T) {
	v := DefaultReadingValidator()

	tests := []struct {
		name        string
		req         *UpdateReadingRequest
		expectError bool
	}{
		{
			name: "valid update with value",
			req: &UpdateReadingRequest{
				Value: float64Ptr(150.0),
			},
			expectError: false,
		},
		{
			name: "valid verify",
			req: &UpdateReadingRequest{
				Verified:   boolPtr(true),
				VerifiedBy: intPtr(1),
			},
			expectError: false,
		},
		{
			name: "verify without verified_by",
			req: &UpdateReadingRequest{
				Verified: boolPtr(true),
			},
			expectError: true,
		},
		{
			name: "negative value",
			req: &UpdateReadingRequest{
				Value: float64Ptr(-10.0),
			},
			expectError: true,
		},
		{
			name: "zero value",
			req: &UpdateReadingRequest{
				Value: float64Ptr(0),
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateUpdate(tt.req)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseDate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expectOK bool
	}{
		{
			name:     "valid date",
			input:    "2024-01-15",
			expectOK: true,
		},
		{
			name:     "empty string",
			input:    "",
			expectOK: true,
		},
		{
			name:     "invalid format",
			input:    "15-01-2024",
			expectOK: false,
		},
		{
			name:     "invalid date",
			input:    "2024-13-45",
			expectOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDate(tt.input)
			if tt.expectOK {
				assert.NoError(t, err)
				if tt.input != "" {
					assert.False(t, result.IsZero())
				}
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func float64Ptr(f float64) *float64 {
	return &f
}

func intPtr(i int) *int {
	return &i
}
