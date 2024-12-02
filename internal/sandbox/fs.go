package sandbox

import (
	"bytes"
	"io/fs"
	"strings"
	"sync"
	"time"
)

const (
	// In WASI, the path separator is always `/`. https://wa.dev/wasi:filesystem
	PathSeparator = '/'
)

var (
	_ fs.FS       = &FS{}
	_ fs.File     = &File{}
	_ fs.FileInfo = &File{}

	root = &File{
		name: "." + string(PathSeparator),
		data: nil,
		ro:   true,
	}
)

func NewFS() *FS {
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

func (f *FS) GetFile(name string) (*File, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	file, ok := f.files[name]
	return file, ok
}

func (f *FS) AddFile(name string, data []byte, ro bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, ok := f.files[name]; ok {
		return fs.ErrExist
	}

	if strings.ContainsRune(name, PathSeparator) {
		return fs.ErrInvalid
	}

	f.files[name] = &File{
		name: name,
		data: bytes.NewBuffer(data),
		ro:   ro,
	}
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

type File struct {
	mu   sync.RWMutex
	name string
	data *bytes.Buffer
	ro   bool
}

func (f *File) Stat() (fs.FileInfo, error) {
	return f, nil
}

func (f *File) Read(p []byte) (n int, err error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return f.data.Read(p)
}

func (f *File) Write(p []byte) (n int, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.ro {
		return 0, fs.ErrPermission
	}

	return f.data.Write(p)
}

func (f *File) Close() error {
	return nil
}

func (f *File) Name() string {
	return f.name
}

func (f *File) Size() int64 {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if f.data == nil {
		return 0
	}

	return int64(f.data.Len())
}

func (f *File) Mode() fs.FileMode {
	if f.IsDir() {
		return fs.ModeDir | 0444
	}

	if f.ro {
		return 0444
	}

	return 0666
}

func (f *File) ModTime() time.Time {
	return time.Time{}
}

func (f *File) IsDir() bool {
	return strings.HasSuffix(f.name, string(PathSeparator))
}

func (f *File) Sys() any {
	return nil
}

func (f *File) String() string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return fs.FormatFileInfo(f)
}
