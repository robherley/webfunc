package sandbox

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/robherley/webfunc-go/internal/virtfs"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/tetratelabs/wazero/sys"
)

type WazeroSandbox struct {
	ctx      context.Context
	runtime  wazero.Runtime
	wasm     []byte
	stdout   *bytes.Buffer
	stderr   *bytes.Buffer
	exitCode uint32
}

func NewWazeroSandbox(ctx context.Context, wasm []byte) (*WazeroSandbox, error) {
	runtime := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigInterpreter())
	_, err := wasi_snapshot_preview1.Instantiate(ctx, runtime)
	if err != nil {
		return nil, err
	}

	return &WazeroSandbox{
		ctx:     ctx,
		runtime: runtime,
		wasm:    wasm,
		stdout:  new(bytes.Buffer),
		stderr:  new(bytes.Buffer),
	}, nil
}

func (s *WazeroSandbox) Handler(w http.ResponseWriter, r *http.Request) error {
	defer s.runtime.Close(s.ctx)
	defer r.Body.Close()

	var err error

	tee := io.MultiWriter(w, s.stdout)

	fsConfig := wazero.NewFSConfig().
		WithFSMount(FS(r), "/var/webfunc")

	cfg := wazero.NewModuleConfig().
		WithEnv("WEBFUNC", "1").
		WithEnv("WEBFUNC_MODE", "HTTP").
		WithEnv("WEBFUNC_DIR", "/var/webfunc").
		WithFSConfig(fsConfig).
		WithStdin(r.Body).
		WithStdout(tee).
		WithStderr(s.stderr).
		WithRandSource(rand.Reader)

	// TODO: add timeout
	_, err = s.runtime.InstantiateWithConfig(s.ctx, s.wasm, cfg)
	if err != nil {
		var exitErr *sys.ExitError
		if errors.As(err, &exitErr) {
			s.exitCode = exitErr.ExitCode()
		} else {
			return err
		}
	}

	return nil
}

func (s *WazeroSandbox) Stdout() string {
	return s.stdout.String()
}

func (s *WazeroSandbox) Stderr() string {
	return s.stderr.String()
}

func (s *WazeroSandbox) ExitCode() uint32 {
	return s.exitCode
}

func FS(r *http.Request) *virtfs.FS {
	fs := virtfs.New()

	headers := strings.Builder{}
	for k, v := range r.Header {
		for _, vv := range v {
			headers.WriteString(k)
			headers.WriteString(": ")
			headers.WriteString(vv)
			headers.WriteRune('\n')
		}
	}

	fs.Add(virtfs.NewFile("headers", []byte(headers.String())))
	fs.Add(virtfs.NewFile("method", []byte(r.Method)))
	fs.Add(virtfs.NewFile("path", []byte(r.URL.Path)))
	fs.Add(virtfs.NewFile("query", []byte(r.URL.Query().Encode())))
	return fs
}
