package sandbox

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"io"
	"net/http"

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

	sandboxFS := NewFS()
	sandboxFS.AddFile("hello.txt", []byte("i'm a fake file in the sandbox"), true)
	sandboxFS.AddFile("status_code", nil, false)

	fsConfig := wazero.NewFSConfig().
		WithFSMount(sandboxFS, "/var/webfunc")

	cfg := wazero.NewModuleConfig().
		WithEnv("WEBFUNC", "1").
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
