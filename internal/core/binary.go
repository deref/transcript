package core

import (
	"unicode"
	"unicode/utf8"
)

// isUnprintable returns true if the rune should be considered unprintable
// for binary detection purposes. This includes invalid UTF-8 sequences
// and control characters, but excludes whitespace.
func isUnprintable(r rune) bool {
	return r == utf8.RuneError || (!unicode.IsPrint(r) && !unicode.IsSpace(r))
}

// isBinary determines if data should be treated as binary output.
// Returns true if the data contains null bytes or >10% unprintable characters.
// Properly handles UTF-8 encoded text.
//
// Uses byte-based threshold approximation for performance: we compare unprintable
// rune count against 10% of total byte count. This enables early termination
// when processing large inputs, though it may be slightly inaccurate for text
// with many multi-byte UTF-8 characters (acceptable trade-off for performance).
func isBinary(data []byte) bool {
	if len(data) == 0 {
		return false
	}

	// Calculate 10% threshold based on byte count approximation.
	// This enables early termination for performance.
	threshold := len(data) / 10
	unprintable := 0

	for len(data) > 0 {
		r, size := utf8.DecodeRune(data)
		data = data[size:]

		// Check for null bytes - instant binary classification.
		if r == 0 {
			return true
		}

		// Check if rune is unprintable for binary detection.
		if isUnprintable(r) {
			unprintable++
			// Early exit if we've exceeded the threshold (>10% of bytes).
			if unprintable > threshold {
				return true
			}
		}
	}

	return false
}
