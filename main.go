package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/robherley/webfunc-go/internal/sandbox"
)

var (
	wasmFile []byte
)

func init() {
	wasmPath := flag.String("file", "/Users/robherley/dev/webfunc-handler/dist/main.wasm", "Path to the wasm file")
	flag.Parse()

	var err error
	wasmFile, err = os.ReadFile(*wasmPath)
	if err != nil {
		log.Fatal(err)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	box, err := sandbox.NewWazeroSandbox(r.Context(), wasmFile)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = box.Handler(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Println("---[ stdout ]------")
	println(box.Stdout())
	fmt.Println("---[ stderr ]------")
	println(box.Stderr())
	fmt.Println("---[ exit code ]---")
	println(box.ExitCode())

	// TODO(robherley): set status code from sandbox fs
}

func main() {
	server := http.Server{
		Addr:    "0.0.0.0:8080",
		Handler: http.HandlerFunc(handler),
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
