package cache

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCacheErrors(t *testing.T) {
	t.Run("ErrCacheMiss", func(t *testing.T) {
		err := ErrCacheMiss
		assert.Error(t, err)
		assert.Equal(t, "cache miss", err.Error())
	})

	t.Run("ErrCacheMiss is consistent", func(t *testing.T) {
		// Verify the error is the same instance
		err1 := ErrCacheMiss
		err2 := ErrCacheMiss
		assert.Equal(t, err1, err2)
		assert.Same(t, err1, err2)
	})
}
