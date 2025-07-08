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

func TestWrapError(t *testing.T) {
	// Test wrapping a real error
	originalErr := fmt.Errorf("something went wrong")
	wrappedErr := WrapError(originalErr)
	
	if wrappedErr == nil {
		t.Error("WrapError returned nil for non-nil error")
	}
	
	// Check that the wrapped error contains the original error
	if !strings.Contains(wrappedErr.Error(), "something went wrong") {
		t.Errorf("WrapError didn't include original error: %v", wrappedErr)
	}
	
	// Test wrapping nil
	if WrapError(nil) != nil {
		t.Error("WrapError should return nil for nil input")
	}
}