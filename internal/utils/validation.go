package utils

import (
	"regexp"
	"strings"
)

var (
	// EmailRegex for email validation
	EmailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	// PhoneRegex for basic phone validation
	PhoneRegex = regexp.MustCompile(`^\+?[1-9]\d{1,14}$`)
)

// IsEmailValid checks if an email is valid
func IsEmailValid(email string) bool {
	email = strings.TrimSpace(email)
	if len(email) == 0 || len(email) > 254 {
		return false
	}
	return EmailRegex.MatchString(email)
}

// IsPhoneValid checks if a phone number is valid (E.164 format)
func IsPhoneValid(phone string) bool {
	phone = strings.TrimSpace(phone)
	return PhoneRegex.MatchString(phone)
}

// SanitizeString removes leading/trailing whitespace
func SanitizeString(s string) string {
	return strings.TrimSpace(s)
}
