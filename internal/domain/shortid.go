package domain

import (
	"crypto/rand"
	"fmt"
)

// ShortIDLength is the length of generated short IDs
const ShortIDLength = 8

// crockfordBase32 is the Crockford Base32 alphabet.
// It excludes I, L, O, U to avoid confusion with 1, 1, 0, and V respectively.
// This gives 32 characters for unambiguous human-readable IDs.
const crockfordBase32 = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// NewShortID8 generates a cryptographically random 8-character ID
// using the Crockford Base32 alphabet.
// With 32^8 = ~1.1 trillion possible combinations, collisions are extremely unlikely.
func NewShortID8() (string, error) {
	return generateShortID(ShortIDLength)
}

// generateShortID generates a random ID of the specified length
// using the Crockford Base32 alphabet and crypto/rand.
func generateShortID(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	result := make([]byte, length)
	for i := 0; i < length; i++ {
		// Use modulo to map byte (0-255) to alphabet index (0-31)
		result[i] = crockfordBase32[bytes[i]%32]
	}

	return string(result), nil
}

// IsShortID returns true if the given ID is an 8-character short ID
// (as opposed to a UUID which is 36 characters with dashes).
func IsShortID(id string) bool {
	return len(id) == ShortIDLength
}
