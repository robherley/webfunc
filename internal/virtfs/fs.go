package virtfs

import (
	"io/fs"
	"strings"
	"sync"
)

const (
	// In WASI, the path separator is always `/`. https://wa.dev/wasi:filesystem
	PathSeparator = '/'
)

var (
	_ fs.FS = &FS{}

	root = NewFile("."+string(PathSeparator), nil)
)

func New() *FS {
	return &FS{
		files: make(map[string]*File),
	}
}

// FS is a simple in-memory file system that allows for read and write access to files.
// It's inspired by embed.FS from the standard library. It should be safe for concurrent use.
// Only top level files are supported, directories are not.
type FS struct {
	mu    sync.RWMutex
	files map[string]*File
}

func (f *FS) Get(name string) (*File, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	file, ok := f.files[name]
	return file, ok
}

func (f *FS) Add(file *File) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, ok := f.files[file.Name()]; ok {
		return fs.ErrExist
	}

	if strings.ContainsRune(file.Name(), PathSeparator) {
		return fs.ErrInvalid
	}

	f.files[file.Name()] = file
	return nil
}

func (f *FS) WriteFile(path string, data []byte, _ fs.FileMode) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	file, ok := f.files[path]
	if !ok {
		return fs.ErrNotExist
	}

	if file.IsDir() {
		return fs.ErrInvalid
	}

	_, err := file.Write(data)
	return err
}

func (f *FS) Open(name string) (fs.File, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if name == "." {
		return root, nil
	}

	if file, ok := f.files[name]; ok {
		return file, nil
	}

	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
}
