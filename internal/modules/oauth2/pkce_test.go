package oauth2

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"
)

// TestPKCEVerification tests the PKCE code challenge verification
func TestPKCEVerification(t *testing.T) {
	// Create a mock service for testing
	service := &OAuth2Service{}

	tests := []struct {
		name               string
		codeVerifier       string
		codeChallenge      string
		codeChallengeMethod string
		wantError          bool
		errorContains      string
	}{
		{
			name:                "Valid S256 PKCE challenge",
			codeVerifier:        "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk",
			codeChallenge:       generateS256Challenge("dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"),
			codeChallengeMethod: "S256",
			wantError:           false,
		},
		{
			name:                "Valid plain PKCE challenge",
			codeVerifier:        "my-plain-verifier-123",
			codeChallenge:       "my-plain-verifier-123",
			codeChallengeMethod: "plain",
			wantError:           false,
		},
		{
			name:                "Invalid S256 challenge - wrong verifier",
			codeVerifier:        "wrong-verifier-value",
			codeChallenge:       generateS256Challenge("dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"),
			codeChallengeMethod: "S256",
			wantError:           true,
			errorContains:       "does not match",
		},
		{
			name:                "Invalid plain challenge - wrong verifier",
			codeVerifier:        "wrong-verifier",
			codeChallenge:       "correct-verifier",
			codeChallengeMethod: "plain",
			wantError:           true,
			errorContains:       "does not match",
		},
		{
			name:                "Missing code verifier",
			codeVerifier:        "",
			codeChallenge:       generateS256Challenge("test"),
			codeChallengeMethod: "S256",
			wantError:           true,
			errorContains:       "required",
		},
		{
			name:                "Minimum length verifier (43 chars)",
			codeVerifier:        "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", // 43 chars
			codeChallenge:       generateS256Challenge("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
			codeChallengeMethod: "S256",
			wantError:           false,
		},
		{
			name:                "Maximum length verifier (128 chars)",
			codeVerifier:        "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", // 128 chars
			codeChallenge:       generateS256Challenge("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
			codeChallengeMethod: "S256",
			wantError:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.verifyPKCE(tt.codeChallenge, tt.codeChallengeMethod, tt.codeVerifier)

			if tt.wantError {
				if err == nil {
					t.Errorf("verifyPKCE() expected error but got none")
					return
				}
				if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
					t.Errorf("verifyPKCE() error = %v, expected to contain %v", err, tt.errorContains)
				}
			} else {
				if err != nil {
					t.Errorf("verifyPKCE() unexpected error = %v", err)
				}
			}
		})
	}
}

// TestPKCEChallengeGeneration tests PKCE challenge generation
func TestPKCEChallengeGeneration(t *testing.T) {
	tests := []struct {
		name         string
		verifier     string
		wantChallenge string // Expected for S256
	}{
		{
			name:         "RFC 7636 Example",
			verifier:     "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk",
			wantChallenge: "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
		},
		{
			name:         "Simple verifier",
			verifier:     "test-verifier",
			wantChallenge: generateS256Challenge("test-verifier"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateS256Challenge(tt.verifier)
			if got != tt.wantChallenge {
				t.Errorf("generateS256Challenge() = %v, want %v", got, tt.wantChallenge)
			}
		})
	}
}

// TestPKCESecurityRequirements tests PKCE security requirements
func TestPKCESecurityRequirements(t *testing.T) {
	service := &OAuth2Service{}

	t.Run("Verifier must be URL-safe", func(t *testing.T) {
		// URL-safe characters: A-Z, a-z, 0-9, -, _, ., ~
		urlSafeVerifier := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
		challenge := generateS256Challenge(urlSafeVerifier)
		err := service.verifyPKCE(challenge, "S256", urlSafeVerifier)
		if err != nil {
			t.Errorf("URL-safe verifier should be valid, got error: %v", err)
		}
	})

	t.Run("Challenge must be deterministic", func(t *testing.T) {
		verifier := "test-verifier-123"
		challenge1 := generateS256Challenge(verifier)
		challenge2 := generateS256Challenge(verifier)

		if challenge1 != challenge2 {
			t.Errorf("Challenge generation not deterministic: %s != %s", challenge1, challenge2)
		}
	})

	t.Run("Different verifiers produce different challenges", func(t *testing.T) {
		challenge1 := generateS256Challenge("verifier1")
		challenge2 := generateS256Challenge("verifier2")

		if challenge1 == challenge2 {
			t.Errorf("Different verifiers produced same challenge")
		}
	})

	t.Run("S256 is more secure than plain", func(t *testing.T) {
		verifier := "my-secret-verifier"
		s256Challenge := generateS256Challenge(verifier)

		// S256 challenge should not reveal the verifier
		if s256Challenge == verifier {
			t.Errorf("S256 challenge should not equal verifier (use hashing)")
		}

		// S256 challenge should be base64url encoded
		_, err := base64.RawURLEncoding.DecodeString(s256Challenge)
		if err != nil {
			t.Errorf("S256 challenge should be valid base64url: %v", err)
		}
	})
}

// TestPKCEEdgeCases tests edge cases in PKCE verification
func TestPKCEEdgeCases(t *testing.T) {
	service := &OAuth2Service{}

	t.Run("Empty challenge with empty verifier", func(t *testing.T) {
		err := service.verifyPKCE("", "S256", "")
		if err == nil {
			t.Error("Empty verifier should fail")
		}
	})

	t.Run("Whitespace in verifier", func(t *testing.T) {
		verifier := "verifier with spaces"
		challenge := generateS256Challenge(verifier)
		err := service.verifyPKCE(challenge, "S256", verifier)
		if err != nil {
			t.Errorf("Verifier with spaces should work: %v", err)
		}
	})

	t.Run("Case sensitivity", func(t *testing.T) {
		verifier := "TestVerifier"
		challenge := generateS256Challenge(verifier)

		// Correct case should work
		err := service.verifyPKCE(challenge, "S256", verifier)
		if err != nil {
			t.Errorf("Correct case should work: %v", err)
		}

		// Wrong case should fail
		err = service.verifyPKCE(challenge, "S256", "testverifier")
		if err == nil {
			t.Error("Wrong case should fail (case sensitive)")
		}
	})

	t.Run("Unicode characters", func(t *testing.T) {
		verifier := "verifier-with-emoji-🎉"
		challenge := generateS256Challenge(verifier)
		err := service.verifyPKCE(challenge, "S256", verifier)
		if err != nil {
			t.Errorf("Unicode verifier should work: %v", err)
		}
	})
}

// Benchmark PKCE verification performance
func BenchmarkPKCEVerification(b *testing.B) {
	service := &OAuth2Service{}
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	challenge := generateS256Challenge(verifier)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.verifyPKCE(challenge, "S256", verifier)
	}
}

func BenchmarkPKCEChallengeGeneration(b *testing.B) {
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generateS256Challenge(verifier)
	}
}

// Helper functions

func generateS256Challenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
