package component

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/lucimux/arya/internal/entity"
	"github.com/lucimux/arya/internal/repo"
)

const (
	_fileExt       = ".json"
	_dirPerm       = 0o750
	_filePerm      = 0o600
	_componentsDir = "components"
)

// Repo -.
type Repo struct {
	schemasDir string
}

// New returns a Component store instrumented with OpenTelemetry tracing spans.
func New(schemasDir string) repo.ComponentStore {
	return newTraced(&Repo{schemasDir: schemasDir})
}

func (r *Repo) projectComponentsDir(projectSlug string) string {
	if projectSlug == "" {
		projectSlug = "default"
	}

	return filepath.Join(r.schemasDir, projectSlug, _componentsDir)
}

func (r *Repo) path(projectSlug, uid string) string {
	return filepath.Join(r.projectComponentsDir(projectSlug), uid+_fileExt)
}

// Exists -.
func (r *Repo) Exists(_ context.Context, projectSlug, uid string) (bool, error) {
	_, err := os.Stat(r.path(projectSlug, uid))
	if err == nil {
		return true, nil
	}

	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}

	return false, fmt.Errorf("ComponentStore - Exists - os.Stat: %w", err)
}

// Save writes the component definition through a temp file and rename.
func (r *Repo) Save(_ context.Context, projectSlug string, c *entity.Component) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("ComponentStore - Save - json.MarshalIndent: %w", err)
	}

	targetDir := r.projectComponentsDir(projectSlug)
	if err = os.MkdirAll(targetDir, _dirPerm); err != nil {
		return fmt.Errorf("ComponentStore - Save - os.MkdirAll: %w", err)
	}

	tmp, err := os.CreateTemp(targetDir, "."+c.UID()+"-*.tmp")
	if err != nil {
		return fmt.Errorf("ComponentStore - Save - os.CreateTemp: %w", err)
	}

	tmpName := tmp.Name()

	_, err = tmp.Write(append(data, '\n'))
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}

	if err == nil {
		err = os.Chmod(tmpName, _filePerm)
	}

	if err == nil {
		err = os.Rename(tmpName, r.path(projectSlug, c.UID()))
	}

	if err != nil {
		return errors.Join(
			fmt.Errorf("ComponentStore - Save - write %s: %w", tmpName, err),
			os.Remove(tmpName),
		)
	}

	return nil
}

// Get -.
func (r *Repo) Get(_ context.Context, projectSlug, uid string) (*entity.Component, error) {
	data, err := os.ReadFile(r.path(projectSlug, uid))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, entity.ErrComponentNotFound
		}

		return nil, fmt.Errorf("ComponentStore - Get - os.ReadFile: %w", err)
	}

	var c entity.Component
	if err = json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("ComponentStore - Get - json.Unmarshal: %w", err)
	}

	return &c, nil
}

// List -.
func (r *Repo) List(_ context.Context, projectSlug string) ([]entity.Component, error) {
	dir := r.projectComponentsDir(projectSlug)

	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}

		return nil, fmt.Errorf("ComponentStore - List - os.ReadDir: %w", err)
	}

	components := make([]entity.Component, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), _fileExt) || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}

		var c entity.Component
		if err = json.Unmarshal(data, &c); err != nil {
			continue
		}

		components = append(components, c)
	}

	return components, nil
}

// Delete -.
func (r *Repo) Delete(_ context.Context, projectSlug, uid string) error {
	err := os.Remove(r.path(projectSlug, uid))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return entity.ErrComponentNotFound
		}

		return fmt.Errorf("ComponentStore - Delete - os.Remove: %w", err)
	}

	return nil
}
