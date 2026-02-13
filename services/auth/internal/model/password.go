package model

import (
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrPasswordTooShort   = errors.New("password must be at least 8 characters")
	ErrPasswordTooLong    = errors.New("password must be at most 128 characters")
	ErrPasswordMissingUpper = errors.New("password must contain at least one uppercase letter")
	ErrPasswordMissingLower = errors.New("password must contain at least one lowercase letter")
	ErrPasswordMissingDigit = errors.New("password must contain at least one digit")
)

// PasswordValidator validates password strength
type PasswordValidator struct {
	RequireMinLength int
	RequireMaxLength int
	RequireUppercase bool
	RequireLowercase bool
	RequireDigit    bool
}

// DefaultPasswordValidator returns a validator with default rules
func DefaultPasswordValidator() *PasswordValidator {
	return &PasswordValidator{
		RequireMinLength: 8,
		RequireMaxLength: 128,
		RequireUppercase: true,
		RequireLowercase: true,
		RequireDigit:    true,
	}
}

// Validate checks if password meets all requirements
func (v *PasswordValidator) Validate(password string) error {
	if len(password) < v.RequireMinLength {
		return ErrPasswordTooShort
	}
	if len(password) > v.RequireMaxLength {
		return ErrPasswordTooLong
	}
	if v.RequireUppercase && !containsUpper(password) {
		return ErrPasswordMissingUpper
	}
	if v.RequireLowercase && !containsLower(password) {
		return ErrPasswordMissingLower
	}
	if v.RequireDigit && !containsDigit(password) {
		return ErrPasswordMissingDigit
	}
	return nil
}

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword checks if a password matches a hash
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func containsUpper(s string) bool {
	for _, c := range s {
		if c >= 'A' && c <= 'Z' {
			return true
		}
	}
	return false
}

func containsLower(s string) bool {
	for _, c := range s {
		if c >= 'a' && c <= 'z' {
			return true
		}
	}
	return false
}

func containsDigit(s string) bool {
	for _, c := range s {
		if c >= '0' && c <= '9' {
			return true
		}
	}
	return false
}

// IsCommonPassword checks against a list of common passwords
func IsCommonPassword(password string) bool {
	commonPasswords := []string{
		"password", "12345678", "qwerty123", "abc12345",
		"password1", "123456789", "qwerty", "12345678",
	}

	lowerPassword := strings.ToLower(password)
	for _, common := range commonPasswords {
		if lowerPassword == common {
			return true
		}
	}
	return false
}
