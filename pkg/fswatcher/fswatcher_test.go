package fswatcher_test

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lucimux/arya/pkg/fswatcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	_debounce = 50 * time.Millisecond
	_settle   = 400 * time.Millisecond
)

func TestWatcherDebouncesJSONChanges(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "schemas") // New must create it

	var calls atomic.Int32

	w, err := fswatcher.New(dir, func() { calls.Add(1) }, fswatcher.Debounce(_debounce))
	require.NoError(t, err)

	w.Start()

	t.Cleanup(func() { require.NoError(t, w.Close()) })

	for range 5 {
		require.NoError(t, os.WriteFile(filepath.Join(dir, "a.json"), []byte("{}"), 0o600))
	}

	assert.Eventually(t, func() bool { return calls.Load() == 1 }, _settle, _debounce/5)

	time.Sleep(_settle / 2)
	assert.Equal(t, int32(1), calls.Load(), "a burst of writes must collapse into one call")
}

func TestWatcherIgnoresOtherExtensions(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	var calls atomic.Int32

	w, err := fswatcher.New(dir, func() { calls.Add(1) }, fswatcher.Debounce(_debounce))
	require.NoError(t, err)

	w.Start()

	t.Cleanup(func() { require.NoError(t, w.Close()) })

	require.NoError(t, os.WriteFile(filepath.Join(dir, ".a-123.tmp"), []byte("{}"), 0o600))

	time.Sleep(_settle)
	assert.Equal(t, int32(0), calls.Load())
}
