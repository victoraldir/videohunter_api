package handlers

import (
	"strings"
	"unicode"
)

// sanitizeText trims user supplied text and drops control characters, which
// have no place in a folder name or a chat message. It does not truncate:
// callers reject input that is too long so nothing is silently cut.
func sanitizeText(raw string) string {
	builder := strings.Builder{}

	for _, character := range raw {
		// Newlines are kept out too: chat messages are single line, and a
		// folder name is one line by definition.
		if unicode.IsControl(character) {
			continue
		}

		builder.WriteRune(character)
	}

	return strings.TrimSpace(builder.String())
}
