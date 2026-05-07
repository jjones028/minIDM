package oauth2

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

// LoadOrGenerateRSAKey loads an RSA private key from path.
// If the file does not exist, it generates a new 2048-bit key and writes it.
func LoadOrGenerateRSAKey(path string) (*rsa.PrivateKey, error) {
	if data, err := os.ReadFile(path); err == nil {
		block, _ := pem.Decode(data)
		if block == nil {
			return nil, fmt.Errorf("oauth2: failed to decode PEM block from %s", path)
		}
		key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("oauth2: parsing RSA key: %w", err)
		}
		return key, nil
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("oauth2: generating RSA key: %w", err)
	}

	data := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	if writeErr := os.WriteFile(path, data, 0600); writeErr != nil {
		fmt.Printf("oauth2: warning: could not persist signing key to %s: %v\n", path, writeErr)
	}
	return key, nil
}

// KeyID returns a stable, short hex string derived from the RSA public key.
func KeyID(key *rsa.PrivateKey) string {
	der, _ := x509.MarshalPKIXPublicKey(&key.PublicKey)
	h := sha256.Sum256(der)
	return fmt.Sprintf("%x", h[:8])
}
