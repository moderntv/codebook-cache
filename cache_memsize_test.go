package codebook

import (
	"context"
	"testing"
	"time"

	"github.com/moderntv/codebook-cache/internal/test_utils"
	"github.com/stretchr/testify/assert"
)

type entryMemTestGeneric struct {
	intValue   int64
	intPointer *int64
	strValue   string
	strPointer *string
}

func testCacheMemsizeCalculated(t *testing.T) {
	t.Parallel()

	c, err := New(Params[string, entryMemTestGeneric]{
		Context: context.Background(),
		Log:     test_utils.Logger(),
		Name:    "testing_cache",
		LoadAllFunc: func(ctx context.Context) (map[string]*entryMemTestGeneric, error) {
			return map[string]*entryMemTestGeneric{
				"key1": {
					intValue:   135,
					intPointer: test_utils.Int64Pointer(89465),
					strValue:   "abcde",
					strPointer: test_utils.StringPointer("abcdefgh jkl"),
				},
				"key2": {
					intValue:   468,
					intPointer: nil,
					strValue:   "abcdefgh ijklmnop qr",
					strPointer: nil,
				},
			}, nil
		},
		MemsizeEnabled: true,
		Timeouts: Timeouts{
			ReloadInterval: 2 * time.Second,
			ReloadDelay:    1 * time.Second,
			Ranomizer:      0,
		},
	})

	assert.NoError(t, err)

	// wait for memsize to be updated
	time.Sleep(time.Second)

	size := c.memSizeValue.Load()
	assert.Greater(t, size, uint64(136))
	assert.Less(t, size, uint64(200))
}

type entryMemTestManual struct{}

func (e *entryMemTestManual) MemSize() uint64 {
	return uint64(120)
}

func TestCacheMemsizeManual(t *testing.T) {
	t.Parallel()

	c, err := New(Params[string, entryMemTestManual]{
		Context: context.Background(),
		Log:     test_utils.Logger(),
		Name:    "testing_cache",
		LoadAllFunc: func(ctx context.Context) (map[string]*entryMemTestManual, error) {
			return map[string]*entryMemTestManual{
				"key1": {},
				"key2": {},
				"key3": {},
				"key4": {},
				"key5": {},
			}, nil
		},
		MemsizeEnabled: true,
		Timeouts: Timeouts{
			ReloadInterval: 2 * time.Second,
			ReloadDelay:    1 * time.Second,
			Ranomizer:      0,
		},
	})

	assert.NoError(t, err)

	// wait for memsize to be updated
	time.Sleep(time.Second)

	size := c.memSizeValue.Load()
	assert.Equal(t, uint64(600), size)
}
