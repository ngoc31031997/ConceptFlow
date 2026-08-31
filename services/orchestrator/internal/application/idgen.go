package application

import (
	"crypto/rand"
	"fmt"
)

// newUUID generates a random UUIDv4 without pulling in an external
// dependency — the module's only external deps are chi/pgx/amqp091-go
// (go.mod), so id generation stays in the standard library.
func newUUID() string {
	b := make([]byte, 16)
	// crypto/rand.Read never returns a non-nil error on supported platforms;
	// a failure here would indicate an unrecoverable OS-level RNG problem.
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("newUUID: failed to read random bytes: %v", err))
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
