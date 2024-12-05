package virtfs

import (
	"bytes"
	"io"
	"io/fs"
	"strings"
	"sync"
	"time"
)

var (
	_ fs.File     = &File{}
	_ fs.FileInfo = &File{}
)

type File struct {
	mu     sync.RWMutex
	name   string
	data   *bytes.Buffer
	writer io.Writer
}

func NewFile(name string, data []byte, opts ...FileOption) *File {
	return &File{
		name:   name,
		data:   bytes.NewBuffer(data),
		writer: nil,
	}
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

	if f.ReadOnly() {
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

	if f.ReadOnly() {
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

func (f *File) ReadOnly() bool {
	return f.writer == nil
}

type FileOption func(*File)

func Writable(w io.Writer) FileOption {
	return func(f *File) {
		f.writer = w
	}
}
