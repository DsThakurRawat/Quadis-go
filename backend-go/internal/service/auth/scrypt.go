package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/scrypt"
)

const (
	// Scrypt parameters matching Node.js crypto.scrypt defaults
	ScryptN      = 16384 // CPU/memory cost parameter (2^14)
	ScryptR      = 8     // Block size parameter
	ScryptP      = 1     // Parallelization parameter
	KeyLen       = 64    // Key length in bytes
	SaltByteSize = 16    // 16 bytes = 32 hex characters
	SchemePrefix = "scrypt"
)

var (
	ErrInvalidHashFormat = errors.New("invalid scrypt hash format")
	ErrSaltGeneration    = errors.New("failed to generate random salt")
)

// HashPassword hashes a plain-text password using scrypt with a random 16-byte
// hex-encoded salt, producing the exact format used by the Node.js backend:
// scrypt$<32-char-hex-salt>$<128-char-hex-derived-key>
func HashPassword(password string) (string, error) {
	saltBytes := make([]byte, SaltByteSize)
	if _, err := rand.Read(saltBytes); err != nil {
		return "", fmt.Errorf("%w: %v", ErrSaltGeneration, err)
	}

	// Node.js does: salt = randomBytes(16).toString('hex')
	// and passes the resulting 32-character hex string as UTF-8 bytes to scrypt.
	saltHex := hex.EncodeToString(saltBytes)

	derived, err := scrypt.Key([]byte(password), []byte(saltHex), ScryptN, ScryptR, ScryptP, KeyLen)
	if err != nil {
		return "", fmt.Errorf("scrypt key derivation failed: %w", err)
	}

	return fmt.Sprintf("%s$%s$%s", SchemePrefix, saltHex, hex.EncodeToString(derived)), nil
}

// VerifyPassword verifies a password against a stored scrypt hash string.
// Returns true if the password matches, false otherwise.
// Uses constant-time comparison to prevent timing attacks.
func VerifyPassword(password string, stored string) bool {
	parts := strings.Split(stored, "$")
	if len(parts) != 3 {
		return false
	}

	scheme, saltHex, hashHex := parts[0], parts[1], parts[2]
	if scheme != SchemePrefix || saltHex == "" || hashHex == "" {
		return false
	}

	expectedHash, err := hex.DecodeString(hashHex)
	if err != nil || len(expectedHash) != KeyLen {
		return false
	}

	// Node passes salt as UTF-8 string to scrypt
	derived, err := scrypt.Key([]byte(password), []byte(saltHex), ScryptN, ScryptR, ScryptP, KeyLen)
	if err != nil {
		return false
	}

	if len(derived) != len(expectedHash) {
		return false
	}

	return subtle.ConstantTimeCompare(derived, expectedHash) == 1
}
