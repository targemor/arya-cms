package fswatcher

import "time"

// Option -.
type Option func(*Watcher)

// Debounce sets how long the directory must be quiet before onChange runs.
func Debounce(d time.Duration) Option {
	return func(w *Watcher) {
		w.debounce = d
	}
}

// Extension sets which files trigger onChange, e.g. ".json".
func Extension(ext string) Option {
	return func(w *Watcher) {
		w.ext = ext
	}
}

// OnError sets a handler for errors reported by the underlying watcher. They are not fatal.
func OnError(f func(error)) Option {
	return func(w *Watcher) {
		w.onError = f
	}
}
