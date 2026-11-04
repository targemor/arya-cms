// Package local implements the "local" Media Library storage provider: files on the server's disk,
// served by the HTTP server under a public URL prefix.
package local

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/lucimux/arya/internal/entity"
	"github.com/lucimux/arya/internal/repo"
)

const (
	_providerName = "local"
	_dirPerm      = 0o750
	_filePerm     = 0o600
)

// Provider -.
type Provider struct {
	dir     string
	baseURL string
}

// New returns a local storage provider that writes into dir (e.g. ./public/uploads) and builds URLs
// under baseURL (e.g. /uploads), instrumented with OpenTelemetry tracing spans.
func New(dir, baseURL string) repo.StorageProvider {
	return newTraced(&Provider{
		dir:     dir,
		baseURL: strings.TrimRight(baseURL, "/"),
	})
}

// Name -.
func (p *Provider) Name() string {
	return _providerName
}

// Upload -.
func (p *Provider) Upload(_ context.Context, file *entity.UploadFile, content io.Reader) error {
	name, err := file.ObjectName()
	if err != nil {
		return fmt.Errorf("LocalStorage - Upload - file.ObjectName: %w", err)
	}

	if err = os.MkdirAll(p.dir, _dirPerm); err != nil {
		return fmt.Errorf("LocalStorage - Upload - os.MkdirAll: %w", err)
	}

	path := filepath.Join(p.dir, name)

	// O_EXCL: a hash collision must fail, not overwrite another file.
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, _filePerm)
	if err != nil {
		return fmt.Errorf("LocalStorage - Upload - os.OpenFile: %w", err)
	}

	_, err = io.Copy(f, content)
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}

	if err != nil {
		// Leave no partial file behind.
		return errors.Join(fmt.Errorf("LocalStorage - Upload - write %s: %w", name, err), os.Remove(path))
	}

	file.URL = p.baseURL + "/" + name
	file.Provider = _providerName

	return nil
}

// Delete -.
func (p *Provider) Delete(_ context.Context, file *entity.UploadFile) error {
	name, err := file.ObjectName()
	if err != nil {
		return fmt.Errorf("LocalStorage - Delete - file.ObjectName: %w", err)
	}

	err = os.Remove(filepath.Join(p.dir, name))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("LocalStorage - Delete - os.Remove: %w", err)
	}

	return nil
}
