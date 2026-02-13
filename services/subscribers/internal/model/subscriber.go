package model

import "time"

// Subscriber represents a water utility customer
type Subscriber struct {
	ID           int       `json:"id" db:"id"`
	AccountNumber string   `json:"account_number" db:"account_number"` // Лицевой счёт
	LastName     string    `json:"last_name" db:"last_name"`
	FirstName    string    `json:"first_name" db:"first_name"`
	MiddleName   *string   `json:"middle_name,omitempty" db:"middle_name"`
	Email        string    `json:"email" db:"email"`
	Phone        string    `json:"phone" db:"phone"`
	Address      string    `json:"address" db:"address"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// CreateSubscriberRequest represents a request to create a new subscriber
type CreateSubscriberRequest struct {
	AccountNumber string `json:"account_number" binding:"required"`
	LastName      string `json:"last_name" binding:"required"`
	FirstName     string `json:"first_name" binding:"required"`
	MiddleName    string `json:"middle_name,omitempty"`
	Email         string `json:"email" binding:"required,email"`
	Phone         string `json:"phone" binding:"required"`
	Address       string `json:"address" binding:"required"`
}

// UpdateSubscriberRequest represents a request to update a subscriber
type UpdateSubscriberRequest struct {
	LastName   *string `json:"last_name,omitempty"`
	FirstName  *string `json:"first_name,omitempty"`
	MiddleName *string `json:"middle_name,omitempty"`
	Email      *string `json:"email,omitempty"`
	Phone      *string `json:"phone,omitempty"`
	Address    *string `json:"address,omitempty"`
}
