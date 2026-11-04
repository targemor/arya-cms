// Package contenttype implements the file-backed ContentType store: one JSON Schema document per
// content type, at <dir>/<projectSlug>/<tableName>.json.
package contenttype

import (
	"context"
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
	_fileExt  = ".json"
	_dirPerm  = 0o750
	_filePerm = 0o600
)

// Repo -.
type Repo struct {
	dir string
}

// New returns a ContentType store instrumented with OpenTelemetry tracing spans.
func New(dir string) repo.ContentTypeStore {
	return newTraced(&Repo{dir: dir})
}

func (r *Repo) projectDir(projectSlug string) string {
	if projectSlug == "" {
		projectSlug = "default"
	}

	return filepath.Join(r.dir, projectSlug)
}

func (r *Repo) path(projectSlug, tableName string) string {
	return filepath.Join(r.projectDir(projectSlug), tableName+_fileExt)
}

// Exists -.
func (r *Repo) Exists(_ context.Context, projectSlug, tableName string) (bool, error) {
	_, err := os.Stat(r.path(projectSlug, tableName))
	if err == nil {
		return true, nil
	}

	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}

	return false, fmt.Errorf("ContentTypeStore - Exists - os.Stat: %w", err)
}

// Save writes the definition through a temp file and a rename.
func (r *Repo) Save(_ context.Context, projectSlug string, ct entity.ContentType) error {
	data, err := marshalSchema(ct)
	if err != nil {
		return fmt.Errorf("ContentTypeStore - Save - marshalSchema: %w", err)
	}

	targetDir := r.projectDir(projectSlug)
	if err = os.MkdirAll(targetDir, _dirPerm); err != nil {
		return fmt.Errorf("ContentTypeStore - Save - os.MkdirAll: %w", err)
	}

	tmp, err := os.CreateTemp(targetDir, "."+ct.TableName+"-*.tmp")
	if err != nil {
		return fmt.Errorf("ContentTypeStore - Save - os.CreateTemp: %w", err)
	}

	tmpName := tmp.Name()

	_, err = tmp.Write(data)
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}

	if err == nil {
		err = os.Chmod(tmpName, _filePerm)
	}

	if err == nil {
		err = os.Rename(tmpName, r.path(projectSlug, ct.TableName))
	}

	if err != nil {
		return errors.Join(
			fmt.Errorf("ContentTypeStore - Save - write %s: %w", tmpName, err),
			os.Remove(tmpName),
		)
	}

	return nil
}

// List -.
func (r *Repo) List(_ context.Context, projectSlug string) ([]entity.ContentType, error) {
	dir := r.projectDir(projectSlug)

	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}

		return nil, fmt.Errorf("ContentTypeStore - List - os.ReadDir: %w", err)
	}

	var (
		cts  []entity.ContentType
		errs []error
	)

	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != _fileExt {
			continue
		}

		ct, loadErr := r.load(dir, e.Name())
		if loadErr != nil {
			errs = append(errs, fmt.Errorf("ContentTypeStore - List - %s: %w", e.Name(), loadErr))

			continue
		}

		cts = append(cts, ct)
	}

	return cts, errors.Join(errs...)
}

// ListProjects scans directories in the schemas root.
func (r *Repo) ListProjects(_ context.Context) ([]string, error) {
	entries, err := os.ReadDir(r.dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}

		return nil, fmt.Errorf("ContentTypeStore - ListProjects - os.ReadDir: %w", err)
	}

	var projects []string

	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			projects = append(projects, e.Name())
		}
	}

	return projects, nil
}

func (r *Repo) load(dir, fileName string) (entity.ContentType, error) {
	data, err := os.ReadFile(filepath.Join(dir, fileName))
	if err != nil {
		return entity.ContentType{}, fmt.Errorf("os.ReadFile: %w", err)
	}

	ct, err := unmarshalSchema(data)
	if err != nil {
		return entity.ContentType{}, err
	}

	if want := strings.TrimSuffix(fileName, _fileExt); ct.TableName != want {
		return entity.ContentType{}, fmt.Errorf("%w: title %q does not match file name %q",
			entity.ErrInvalidContentType, ct.TableName, want)
	}

	if err = ct.Validate(); err != nil {
		return entity.ContentType{}, fmt.Errorf("ct.Validate: %w", err)
	}

	return ct, nil
}
