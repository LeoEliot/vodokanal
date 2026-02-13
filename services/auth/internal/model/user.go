package model

import "time"

// User represents a user in the system
type User struct {
	ID           int       `json:"id" db:"id"`
	SubscriberID *int      `json:"subscriber_id,omitempty" db:"subscriber_id"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"`
	Role         string    `json:"role" db:"role"`
	IsActive     bool      `json:"is_active" db:"is_active"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// RefreshToken represents a stored refresh token
type RefreshToken struct {
	ID        int       `json:"id" db:"id"`
	UserID    int       `json:"user_id" db:"user_id"`
	Token     string    `json:"token" db:"token"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name" binding:"required"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// RefreshTokenRequest represents a refresh token request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RegisterResponse represents a successful registration response
type RegisterResponse struct {
	UserID      int    `json:"user_id"`
	Email       string `json:"email"`
	Role        string `json:"role"`
	AccessToken string `json:"access_token"`
}

// LoginResponse represents a successful login response
type LoginResponse struct {
	UserID       int    `json:"user_id"`
	Email        string `json:"email"`
	Role         string `json:"role"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// RefreshResponse represents a successful token refresh response
type RefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// UserResponse represents user information response
type UserResponse struct {
	ID           int        `json:"id"`
	SubscriberID *int       `json:"subscriber_id,omitempty"`
	Email        string     `json:"email"`
	Role         string     `json:"role"`
	IsActive     bool       `json:"is_active"`
	CreatedAt    time.Time  `json:"created_at"`
}

// ToResponse converts User to UserResponse
func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:           u.ID,
		SubscriberID: u.SubscriberID,
		Email:        u.Email,
		Role:         u.Role,
		IsActive:     u.IsActive,
		CreatedAt:    u.CreatedAt,
	}
}
