package auth

import (
	_ "embed"
	"encoding/json"
	"strings"
	"testing"
)

//go:embed testdata/scrypt_vectors.json
var nodeVectorsJSON []byte

type TestVector struct {
	Password   string `json:"password"`
	Salt       string `json:"salt"`
	DerivedHex string `json:"derived_hex"`
	Stored     string `json:"stored"`
}

// TestNodeCompatibility_VerifyNodeHashesInGo tests that hashes produced by
// Node.js crypto.scrypt verify successfully in Go, matching byte-for-byte.
func TestNodeCompatibility_VerifyNodeHashesInGo(t *testing.T) {
	var vectors []TestVector
	if err := json.Unmarshal(nodeVectorsJSON, &vectors); err != nil {
		t.Fatalf("failed to unmarshal test vectors: %v", err)
	}

	if len(vectors) == 0 {
		t.Fatal("no test vectors loaded")
	}

	for _, tc := range vectors {
		tc := tc
		testName := tc.Password
		if testName == "" {
			testName = "<empty-password>"
		}

		t.Run(testName, func(t *testing.T) {
			// 1. Valid password verification must succeed
			if !VerifyPassword(tc.Password, tc.Stored) {
				t.Errorf("expected VerifyPassword to succeed for password: %q", tc.Password)
			}

			// 2. Wrong password must fail
			if VerifyPassword(tc.Password+"_wrong", tc.Stored) {
				t.Errorf("expected VerifyPassword to fail for incorrect password")
			}

			// 3. Altered salt must fail
			parts := strings.Split(tc.Stored, "$")
			if len(parts) == 3 {
				corruptedSalt := "0000" + parts[1][4:]
				corruptedStored := parts[0] + "$" + corruptedSalt + "$" + parts[2]
				if VerifyPassword(tc.Password, corruptedStored) {
					t.Errorf("expected VerifyPassword to fail with altered salt")
				}
			}

			// 4. Altered hash must fail
			if len(parts) == 3 {
				corruptedHash := "ffff" + parts[2][4:]
				corruptedStored := parts[0] + "$" + parts[1] + "$" + corruptedHash
				if VerifyPassword(tc.Password, corruptedStored) {
					t.Errorf("expected VerifyPassword to fail with altered hash")
				}
			}
		})
	}
}

// TestGoHashingFormat verifies that Go-generated hashes follow the exact scrypt format
// and verify correctly with VerifyPassword.
func TestGoHashingFormat(t *testing.T) {
	passwords := []string{
		"SimplePass123",
		"admin-pin-9999",
		"Complex!@#$%^&*()_+=~",
		"",
		"hindi-होटल-2026",
	}

	for _, pw := range passwords {
		pw := pw
		t.Run(pw, func(t *testing.T) {
			hash, err := HashPassword(pw)
			if err != nil {
				t.Fatalf("HashPassword failed: %v", err)
			}

			// Format check: scrypt$<32-char-salt>$<128-char-hash>
			parts := strings.Split(hash, "$")
			if len(parts) != 3 {
				t.Fatalf("expected 3 parts in hash, got %d", len(parts))
			}
			if parts[0] != SchemePrefix {
				t.Errorf("expected scheme %q, got %q", SchemePrefix, parts[0])
			}
			if len(parts[1]) != 32 {
				t.Errorf("expected 32 hex char salt, got %d", len(parts[1]))
			}
			if len(parts[2]) != 128 { // 64 bytes = 128 hex chars
				t.Errorf("expected 128 hex char derived key, got %d", len(parts[2]))
			}

			// Must verify with correct password
			if !VerifyPassword(pw, hash) {
				t.Errorf("VerifyPassword failed for Go-generated hash")
			}

			// Must fail with incorrect password
			if VerifyPassword(pw+"wrong", hash) {
				t.Errorf("VerifyPassword succeeded for wrong password on Go-generated hash")
			}
		})
	}
}

// TestMalformedStoredHashes checks edge cases and invalid formats to ensure safe handling
// without panics.
func TestMalformedStoredHashes(t *testing.T) {
	malformedCases := []struct {
		name   string
		stored string
	}{
		{"empty string", ""},
		{"single component", "scrypt"},
		{"two components", "scrypt$0123456789abcdef0123456789abcdef"},
		{"wrong scheme", "bcrypt$0123456789abcdef0123456789abcdef$8e072ac4"},
		{"empty salt", "scrypt$$8e072ac4"},
		{"empty hash", "scrypt$0123456789abcdef0123456789abcdef$"},
		{"non-hex hash", "scrypt$0123456789abcdef0123456789abcdef$not_hex_characters_here!!"},
		{"truncated hash", "scrypt$0123456789abcdef0123456789abcdef$8e072ac4"},
		{"four parts", "scrypt$salt$hash$extra"},
	}

	for _, tc := range malformedCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if VerifyPassword("anyPassword", tc.stored) {
				t.Errorf("expected VerifyPassword to return false for malformed input %q", tc.stored)
			}
		})
	}
}
