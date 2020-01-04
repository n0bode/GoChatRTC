package chat

import (
	"crypto/sha256"
	"encoding/hex"
)

func EncodeToSha(str string) string {
	buffer := sha256.Sum256([]byte(str))
	return hex.EncodeToString(buffer[:])
}
