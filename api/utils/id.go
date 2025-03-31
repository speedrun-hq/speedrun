package utils

import (
	"github.com/google/uuid"
)

// GenerateID creates a new UUID v4
func GenerateID() string {
	return uuid.New().String()
}
