package validator

import (
	"fmt"
	"regexp"
	"unicode"
)

// ValidateAccountNumber validates Russian account number format
func ValidateAccountNumber(accountNumber string) error {
	if len(accountNumber) < 3 || len(accountNumber) > 20 {
		return fmt.Errorf("account number must be between 3 and 20 characters")
	}

	matched, err := regexp.MatchString(`^[A-Z0-9-]+$`, accountNumber)
	if err != nil || !matched {
		return fmt.Errorf("account number can only contain uppercase letters, numbers and hyphens")
	}

	return nil
}

// ValidateEmail validates email format
func ValidateEmail(email string) error {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

// ValidatePhone validates Russian phone number format
func ValidatePhone(phone string) error {
	// Remove common separators
	cleaned := regexp.MustCompile(`[+\s\-\(\)]`).ReplaceAllString(phone, "")

	if len(cleaned) < 10 || len(cleaned) > 11 {
		return fmt.Errorf("invalid phone number format")
	}

	return nil
}

// ValidateName validates name (Cyrillic or Latin letters)
func ValidateName(name string) error {
	if len(name) < 2 || len(name) > 100 {
		return fmt.Errorf("name must be between 2 and 100 characters")
	}

	for _, r := range name {
		if !unicode.IsLetter(r) && r != '-' && r != ' ' {
			return fmt.Errorf("name can only contain letters, hyphens and spaces")
		}
	}

	return nil
}

// ValidateAddress validates address
func ValidateAddress(address string) error {
	if len(address) < 5 || len(address) > 500 {
		return fmt.Errorf("address must be between 5 and 500 characters")
	}
	return nil
}
