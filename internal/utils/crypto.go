package utils

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
)

// RSAKeyPair holds an RSA private/public key pair
type RSAKeyPair struct {
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
	KeyID      string
}

// GenerateRSAKeyPair generates a new RSA key pair for RS256 signing
func GenerateRSAKeyPair(keyID string, bits int) (*RSAKeyPair, error) {
	if bits < 2048 {
		bits = 2048 // Minimum secure key size
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}

	return &RSAKeyPair{
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
		KeyID:      keyID,
	}, nil
}

// LoadRSAKeyPairFromFiles loads RSA keys from PEM files
func LoadRSAKeyPairFromFiles(privateKeyPath, publicKeyPath, keyID string) (*RSAKeyPair, error) {
	// Read private key
	privateKeyPEM, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	privateBlock, _ := pem.Decode(privateKeyPEM)
	if privateBlock == nil {
		return nil, fmt.Errorf("failed to decode private key PEM")
	}

	var privateKey *rsa.PrivateKey
	if privateBlock.Type == "RSA PRIVATE KEY" {
		privateKey, err = x509.ParsePKCS1PrivateKey(privateBlock.Bytes)
	} else if privateBlock.Type == "PRIVATE KEY" {
		key, err := x509.ParsePKCS8PrivateKey(privateBlock.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse PKCS8 private key: %w", err)
		}
		var ok bool
		privateKey, ok = key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("not an RSA private key")
		}
	} else {
		return nil, fmt.Errorf("unknown private key type: %s", privateBlock.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return &RSAKeyPair{
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
		KeyID:      keyID,
	}, nil
}

// SaveToFiles saves the key pair to PEM files
func (kp *RSAKeyPair) SaveToFiles(privateKeyPath, publicKeyPath string) error {
	// Save private key
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(kp.PrivateKey),
	}

	privateFile, err := os.Create(privateKeyPath)
	if err != nil {
		return fmt.Errorf("failed to create private key file: %w", err)
	}
	defer privateFile.Close()

	if err := pem.Encode(privateFile, privateKeyPEM); err != nil {
		return fmt.Errorf("failed to encode private key: %w", err)
	}

	// Save public key
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(kp.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to marshal public key: %w", err)
	}

	publicKeyPEM := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}

	publicFile, err := os.Create(publicKeyPath)
	if err != nil {
		return fmt.Errorf("failed to create public key file: %w", err)
	}
	defer publicFile.Close()

	if err := pem.Encode(publicFile, publicKeyPEM); err != nil {
		return fmt.Errorf("failed to encode public key: %w", err)
	}

	return nil
}

// JWK represents a JSON Web Key for JWKS endpoint
type JWK struct {
	KeyType   string `json:"kty"`            // Key Type (RSA)
	Use       string `json:"use"`            // Public Key Use (sig)
	KeyID     string `json:"kid"`            // Key ID
	Algorithm string `json:"alg"`            // Algorithm (RS256)
	Modulus   string `json:"n"`              // RSA Modulus (base64url)
	Exponent  string `json:"e"`              // RSA Exponent (base64url)
}

// JWKS represents a JSON Web Key Set
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// ToJWK converts the RSA public key to JWK format
func (kp *RSAKeyPair) ToJWK() JWK {
	return JWK{
		KeyType:   "RSA",
		Use:       "sig",
		KeyID:     kp.KeyID,
		Algorithm: "RS256",
		Modulus:   base64.RawURLEncoding.EncodeToString(kp.PublicKey.N.Bytes()),
		Exponent:  base64.RawURLEncoding.EncodeToString(big.NewInt(int64(kp.PublicKey.E)).Bytes()),
	}
}

// GenerateRandomString generates a cryptographically secure random string
func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[num.Int64()]
	}
	return string(b)
}

// HashSHA256 returns the SHA256 hash of the input string as a hex string
func HashSHA256(input string) string {
	hash := sha256.Sum256([]byte(input))
	return fmt.Sprintf("%x", hash)
}
