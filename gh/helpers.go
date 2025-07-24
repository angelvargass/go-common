package gh

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"

	"golang.org/x/crypto/nacl/box"
)

const (
	keySize   = 32
	nonceSize = 24
)

// encryptSecret encrypts a secret value using sealed box encryption.
//
// It uses the public key of the repository to encrypt the secret.
// The encrypted value is returned as a base64 encoded string.
func (gh *Github) encryptSecret(publicKey, text string) (string, error) {
	// Decode the public key from base64
	publicKeyBytes, err := base64.StdEncoding.DecodeString(publicKey)
	if err != nil {
		gh.Logger.Debug("error decoding public key")
		return "", fmt.Errorf("error decoding public key: %w", err)
	} else if size := len(publicKeyBytes); size != keySize {
		gh.Logger.Debug("recipient public key has invalid length", slog.Int("length", size))
		return "", fmt.Errorf("recipient public key has invalid length (%d bytes)", size)
	}

	// Decode the public key
	var publicKeyDecoded [32]byte
	copy(publicKeyDecoded[:], publicKeyBytes)

	// Encrypt the secret value
	encrypted, err := box.SealAnonymous(nil, []byte(text), (*[32]byte)(publicKeyBytes), rand.Reader)

	if err != nil {
		gh.Logger.Debug("error sealing secret")
		return "", fmt.Errorf("error sealing secret: %w", err)
	}
	// Encode the encrypted value in base64
	encryptedBase64 := base64.StdEncoding.EncodeToString(encrypted)

	return encryptedBase64, nil
}
