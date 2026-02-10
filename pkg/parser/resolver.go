package parser

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ErrNotSupported indicates that a resolver does not handle the given source.
var ErrNotSupported = errors.New("source not supported by this resolver")

// Resolver opens include sources for reading. Implementations return the
// content reader, the resolved absolute path (used for cycle detection and
// relative include resolution), and any error.
type Resolver interface {
	Resolve(source string, basePath string) (io.ReadCloser, string, error)
}

// FileResolver resolves includes from the local filesystem.
type FileResolver struct{}

// Resolve opens a local file relative to basePath and returns its reader and
// absolute path.
func (*FileResolver) Resolve(source string, basePath string) (io.ReadCloser, string, error) {
	abs := source
	if !filepath.IsAbs(source) {
		abs = filepath.Join(basePath, source)
	}
	abs = filepath.Clean(abs)

	f, err := os.Open(abs)
	if err != nil {
		return nil, "", fmt.Errorf("resolving include %q from %s: %w", source, basePath, err)
	}
	return f, abs, nil
}
