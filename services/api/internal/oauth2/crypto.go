package oauth2

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// argon2id parameters for client secrets (same as identity package).
const (
	argonMemory      = 64 * 1024
	argonIterations  = 3
	argonParallelism = 2
	argonSaltLen     = 16
	argonKeyLen      = 32
)

// HashClientSecret hashes a client secret using Argon2id.
func HashClientSecret(secret string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(secret), salt, argonIterations, argonMemory, argonParallelism, argonKeyLen)
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonIterations, argonParallelism, b64Salt, b64Hash), nil
}

// VerifyClientSecret verifies a plain secret against an Argon2id hash.
func VerifyClientSecret(secret, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, fmt.Errorf("oauth2: invalid hash format")
	}
	var memory, iterations uint32
	var parallelism uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
		return false, fmt.Errorf("oauth2: invalid hash params: %w", err)
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("oauth2: invalid salt: %w", err)
	}
	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("oauth2: invalid hash: %w", err)
	}
	computed := argon2.IDKey([]byte(secret), salt, iterations, memory, parallelism, uint32(len(hash)))
	return subtle.ConstantTimeCompare(computed, hash) == 1, nil
}

// HashRefreshToken returns a hex SHA-256 of the token.
// Refresh tokens are already high-entropy random values, so a fast hash is fine.
func HashRefreshToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", h[:])
}

// ComputeCodeChallenge computes the S256 PKCE code_challenge from a verifier.
func ComputeCodeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// generateRandomBase64 returns n random bytes encoded as base64url (no padding).
func generateRandomBase64(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// GenerateClientID returns a short, URL-safe random identifier.
func GenerateClientID() (string, error) {
	return generateRandomBase64(16)
}

// GenerateClientSecret returns a high-entropy random secret (plaintext, shown once).
func GenerateClientSecret() (string, error) {
	return generateRandomBase64(32)
}

// GenerateAuthCode returns a random authorization code.
func GenerateAuthCode() (string, error) {
	return generateRandomBase64(32)
}

// GenerateRefreshToken returns a random refresh token.
func GenerateRefreshToken() (string, error) {
	return generateRandomBase64(32)
}
