package filesystem

import (
	"io/fs"
	"os"
	"path/filepath"
)

// osFS wraps functions working with os filesystem to implement fs.FS interfaces.
type OsFS struct{}

func (OsFS) Open(name string) (fs.File, error) { return os.Open(filepath.FromSlash(name)) }

func (OsFS) ReadDir(name string) ([]fs.DirEntry, error) { return os.ReadDir(filepath.FromSlash(name)) }

func (OsFS) Stat(name string) (fs.FileInfo, error) { return os.Stat(filepath.FromSlash(name)) }

func (OsFS) ReadFile(name string) ([]byte, error) { return os.ReadFile(filepath.FromSlash(name)) }

func (OsFS) Glob(pattern string) ([]string, error) { return filepath.Glob(filepath.FromSlash(pattern)) }
