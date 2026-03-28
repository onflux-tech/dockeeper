package channel

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// NewRequestID generates a random 16-byte hex request ID for command multiplexing.
func NewRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback — should never happen.
		return fmt.Sprintf("fallback-%d", b[0])
	}
	return hex.EncodeToString(b)
}
