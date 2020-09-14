package mux

import "fmt"

// NoLoaderFoundError indicates the url multiplexer was unable to match the url to given auth loader.
type NoLoaderFoundError struct {
	URL string
}

// Error returns a descriptive error message.
func (e NoLoaderFoundError) Error() string {
	return fmt.Sprintf("no loader found for %q", e.URL)
}

// IsNoLoaderFound indicates if the error is of type NoLoaderFoundError.
func IsNoLoaderFound(err error) bool {
	_, ok := err.(NoLoaderFoundError)
	return ok
}
