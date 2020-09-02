package mux

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNoLoaderFoundError(t *testing.T) {
	err := NoLoaderFoundError{URL: "test.com"}
	assert.EqualError(t, err, `no loader found for "test.com"`)
}

func TestIsNoLoaderFound(t *testing.T) {
	assert.True(t, IsNoLoaderFound(NoLoaderFoundError{}))
	assert.False(t, IsNoLoaderFound(errors.New("")))
}
