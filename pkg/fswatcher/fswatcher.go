// Package fswatcher calls a function when files with a given extension change in a directory.
// Bursts of events (an editor saving, a temp file renamed into place) are debounced into one call.
package fswatcher

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

const (
	_defaultDebounce = 500 * time.Millisecond
	_defaultExt      = ".json"
	_dirPerm         = 0o750
)

// Watcher -.
type Watcher struct {
	fsw      *fsnotify.Watcher
	onChange func()
	onError  func(error)
	debounce time.Duration
	ext      string

	done chan struct{}
	wg   sync.WaitGroup
}

// New creates dir if it is missing and starts watching it. onChange runs on the watcher's own
// goroutine, never concurrently with itself.
func New(dir string, onChange func(), opts ...Option) (*Watcher, error) {
	w := &Watcher{
		onChange: onChange,
		onError:  func(error) {},
		debounce: _defaultDebounce,
		ext:      _defaultExt,
		done:     make(chan struct{}),
	}

	// Custom options
	for _, opt := range opts {
		opt(w)
	}

	if err := os.MkdirAll(dir, _dirPerm); err != nil {
		return nil, fmt.Errorf("fswatcher - New - os.MkdirAll: %w", err)
	}

	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("fswatcher - New - fsnotify.NewWatcher: %w", err)
	}

	if err = fsw.Add(dir); err != nil {
		return nil, errors.Join(fmt.Errorf("fswatcher - New - fsw.Add: %w", err), fsw.Close())
	}

	w.fsw = fsw

	return w, nil
}

// Start -.
func (w *Watcher) Start() {
	w.wg.Go(w.run)
}

// Close stops the watcher and waits for an in-flight onChange to return.
func (w *Watcher) Close() error {
	close(w.done)

	err := w.fsw.Close()

	w.wg.Wait()

	if err != nil {
		return fmt.Errorf("fswatcher - Close - fsw.Close: %w", err)
	}

	return nil
}

func (w *Watcher) run() {
	timer := time.NewTimer(w.debounce)
	timer.Stop()

	defer timer.Stop()

	for {
		select {
		case <-w.done:
			return
		case ev, ok := <-w.fsw.Events:
			if !ok {
				return
			}

			if filepath.Ext(ev.Name) == w.ext {
				timer.Reset(w.debounce)
			}
		case err, ok := <-w.fsw.Errors:
			if !ok {
				return
			}

			w.onError(err)
		case <-timer.C:
			w.onChange()
		}
	}
}
