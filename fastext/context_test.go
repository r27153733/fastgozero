package fastext

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testKeyType int

const testKey testKeyType = iota

func TestGetCtxKeyValue(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, testKey, "test")
	kv := getCtxKeyValue(ctx)
	assert.Equal(t, testKey, kv.key)
	assert.Equal(t, "test", kv.val)
}
