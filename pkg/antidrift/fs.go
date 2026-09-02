package antidrift

import (
	"io/fs"
	"os"
)

// FileSystem abstracts disk IO so Verify/Lock can run against mocks.
type FileSystem interface {
	ReadFile(name string) ([]byte, error)
	WriteFile(name string, data []byte, perm fs.FileMode) error
	Stat(name string) (fs.FileInfo, error)
}

// OSFileSystem is the default real-disk implementation.
type OSFileSystem struct{}

func (OSFileSystem) ReadFile(name string) ([]byte, error) { return os.ReadFile(name) }

func (OSFileSystem) WriteFile(name string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(name, data, perm)
}

func (OSFileSystem) Stat(name string) (fs.FileInfo, error) { return os.Stat(name) }

func (e *Engine) fileSystem() FileSystem {
	if e.FS != nil {
		return e.FS
	}
	return OSFileSystem{}
}
