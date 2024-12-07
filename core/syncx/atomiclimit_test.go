package syncx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAtomicLimit(t *testing.T) {
	limit := NewAtomicLimit(2)
	assert.True(t, limit.TryBorrow())
	assert.True(t, limit.TryBorrow())
	assert.False(t, limit.TryBorrow())
	limit.Return()
	assert.Equal(t, limit.m, uint32(1))
	limit.Return()
	assert.Equal(t, limit.m, uint32(0))
}
