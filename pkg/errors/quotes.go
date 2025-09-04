package errors

import (
	"fmt"
	"math/rand"
	"os"
)

var lebowskiQuotes = []string{
	"Smokey, this isn't 'Nam. This is bowling. There are rules.",
	"You're out of your element, Donny!",
	"This aggression will not stand, man!",
	"Obviously you're not a golfer.",
	"Calmer than you are dude.",
}

// No need for init() - Go 1.20+ auto-seeds rand

// LebowskiError returns a random Big Lebowski quote as an error message
func LebowskiError() string {
	return lebowskiQuotes[rand.Intn(len(lebowskiQuotes))]
}

// PrintError prints an error with a Lebowski quote to stderr
func PrintError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "🎳 %s\n", LebowskiError())
	}
}
