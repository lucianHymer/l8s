package errors

import (
	"fmt"
	"math/rand"
	"os"
	"os/user"
	"strings"
)

var lebowskiQuotes = []string{
	"Am I the only one around here who gives a shit about the rules?!",
	"You're out of your element, %s!",
	"This aggression will not stand, man",
	"You're entering a world of pain",
	"Obviously you're not a golfer",
	"Do you see what happens, Larry?",
	"Calmer than you are",
}

// LebowskiError returns a random Big Lebowski quote as an error message
func LebowskiError() string {
	quote := lebowskiQuotes[rand.Intn(len(lebowskiQuotes))]
	
	// If the quote has a %s placeholder, try to insert the username
	if strings.Contains(quote, "%s") {
		username := "Donny" // default fallback
		if u, err := user.Current(); err == nil {
			username = u.Username
		}
		quote = fmt.Sprintf(quote, username)
	}
	
	return quote
}

// WrapError wraps an error with a random Lebowski quote
func WrapError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", LebowskiError(), err)
}

// PrintError prints an error with a Lebowski quote to stderr
func PrintError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "🎳 %s\n", LebowskiError())
		fmt.Fprintf(os.Stderr, "   %v\n", err)
	}
}