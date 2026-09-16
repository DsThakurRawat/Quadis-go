package repository

import (
	"crypto/rand"
	"fmt"
)

// Crockford base32 alphabet minus confusing characters (I, L, O, U)
const codeAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// GenerateBookingCode produces a Crockford base32 booking code with "QD-" prefix
// e.g. "QD-7K9MNP2X" matching the Node.js implementation exactly.
func GenerateBookingCode() (string, error) {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to read random bytes for booking code: %w", err)
	}

	out := make([]byte, 8)
	for i := 0; i < 8; i++ {
		out[i] = codeAlphabet[int(bytes[i])%len(codeAlphabet)]
	}

	return fmt.Sprintf("QD-%s", string(out)), nil
}
