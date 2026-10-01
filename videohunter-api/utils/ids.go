package utils

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"time"
)

// NewId returns a short, URL safe, random identifier. It is used for folder
// ids and chat message ids, which only have to be unique, not unguessable.
func NewId() string {
	buffer := make([]byte, 12)

	if _, err := rand.Read(buffer); err != nil {
		// crypto/rand does not fail in practice; falling back to a fixed
		// value would be worse than failing loudly.
		slog.Error("could not generate an id", "error", err)
		panic(err)
	}

	return base64.RawURLEncoding.EncodeToString(buffer)
}

// NewSortableId returns an id that also sorts chronologically as a string: a
// fixed width millisecond stamp followed by a random suffix. Chat messages use
// it as the tail of their sort key, which is what puts a room's messages in
// reading order.
func NewSortableId(now time.Time) string {
	return fmt.Sprintf("%013d#%s", now.UnixMilli(), NewId())
}
