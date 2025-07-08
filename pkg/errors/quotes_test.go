package errors

import (
	"fmt"
	"strings"
	"testing"
)

func TestLebowskiError(t *testing.T) {
	// Test that we get a valid quote
	quote := LebowskiError()
	if quote == "" {
		t.Error("LebowskiError returned empty string")
	}
	
	// Test that quotes are from our list
	found := false
	for _, q := range lebowskiQuotes {
		// Check if the quote matches (accounting for username substitution)
		if strings.Contains(q, "%s") {
			// For quotes with placeholders, just check the non-placeholder parts
			parts := strings.Split(q, "%s")
			if len(parts) == 2 && strings.Contains(quote, parts[0]) && strings.Contains(quote, parts[1]) {
				found = true
				break
			}
		} else if quote == q {
			found = true
			break
		}
	}
	
	if !found {
		t.Errorf("LebowskiError returned unexpected quote: %s", quote)
	}
}
