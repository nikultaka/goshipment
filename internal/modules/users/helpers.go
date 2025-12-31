package users

import (
	"crypto/rand"
	"encoding/hex"
)

func generateSessionID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
