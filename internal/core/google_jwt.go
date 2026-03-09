package core

// buildGoogleJWT creates a signed RS256 JWT for Google service account authentication.
// Uses only stdlib — no external OAuth2 libraries required.

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"strings"
)

func buildGoogleJWT(privateKeyPEM string, claimsJSON []byte) (string, error) {
	// Parse RSA private key
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return "", fmt.Errorf("failed to decode PEM block for Google service account key")
	}

	var privateKey *rsa.PrivateKey
	switch block.Type {
	case "RSA PRIVATE KEY":
		key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return "", fmt.Errorf("parse PKCS1 key: %w", err)
		}
		privateKey = key
	case "PRIVATE KEY":
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return "", fmt.Errorf("parse PKCS8 key: %w", err)
		}
		var ok bool
		privateKey, ok = key.(*rsa.PrivateKey)
		if !ok {
			return "", fmt.Errorf("service account key is not RSA")
		}
	default:
		return "", fmt.Errorf("unsupported key type: %s", block.Type)
	}

	// Header: {"alg":"RS256","typ":"JWT"}
	headerJSON, _ := json.Marshal(map[string]string{"alg": "RS256", "typ": "JWT"})
	header := base64.RawURLEncoding.EncodeToString(headerJSON)
	payload := base64.RawURLEncoding.EncodeToString(claimsJSON)

	// Signing input
	signingInput := header + "." + payload
	h := sha256.New()
	h.Write([]byte(signingInput))
	digest := h.Sum(nil)

	// Sign
	sig, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, digest)
	if err != nil {
		return "", fmt.Errorf("sign JWT: %w", err)
	}

	return strings.Join([]string{header, payload, base64.RawURLEncoding.EncodeToString(sig)}, "."), nil
}
