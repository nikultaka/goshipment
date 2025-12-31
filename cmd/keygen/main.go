package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"gogin/internal/utils"
)

func main() {
	keyID := flag.String("kid", "default-key-2025", "Key ID for the RSA key pair")
	bits := flag.Int("bits", 2048, "RSA key size in bits (minimum 2048)")
	outputDir := flag.String("output", "./keys", "Output directory for key files")
	flag.Parse()

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(*outputDir, 0700); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	privateKeyPath := filepath.Join(*outputDir, "private_key.pem")
	publicKeyPath := filepath.Join(*outputDir, "public_key.pem")

	// Check if keys already exist
	if _, err := os.Stat(privateKeyPath); err == nil {
		log.Printf("Warning: Keys already exist at %s", *outputDir)
		log.Printf("Delete existing keys to generate new ones")
		return
	}

	// Generate new RSA key pair
	log.Printf("Generating RSA-%d key pair with Key ID: %s", *bits, *keyID)
	keyPair, err := utils.GenerateRSAKeyPair(*keyID, *bits)
	if err != nil {
		log.Fatalf("Failed to generate key pair: %v", err)
	}

	// Save keys to files
	if err := keyPair.SaveToFiles(privateKeyPath, publicKeyPath); err != nil {
		log.Fatalf("Failed to save keys: %v", err)
	}

	log.Printf("✓ RSA key pair generated successfully!")
	log.Printf("  Private key: %s", privateKeyPath)
	log.Printf("  Public key:  %s", publicKeyPath)
	log.Printf("  Key ID:      %s", *keyID)
	log.Printf("\nAdd these to your .env file:")
	log.Printf("  JWT_ALGORITHM=RS256")
	log.Printf("  JWT_KID=%s", *keyID)
	log.Printf("  JWT_PRIVATE_KEY_PATH=%s", privateKeyPath)
	log.Printf("  JWT_PUBLIC_KEY_PATH=%s", publicKeyPath)
}
