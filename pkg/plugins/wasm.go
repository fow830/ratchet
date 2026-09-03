// Package plugins runs WASM fitness rule packs via wazero (no .so plugins).
package plugins

import (
	"context"
	"fmt"
	"os"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

const exportRun = "run"

// Result is the outcome of a WASM plugin run.
type Result struct {
	Code uint32 `json:"code"`
	Name string `json:"name,omitempty"`
}

// Engine executes WASM plugins in an isolated wazero runtime.
type Engine struct {
	rt wazero.Runtime
}

// NewEngine creates a wazero runtime for plugin execution.
func NewEngine(ctx context.Context) (*Engine, error) {
	rt := wazero.NewRuntime(ctx)
	return &Engine{rt: rt}, nil
}

// Close releases runtime resources.
func (e *Engine) Close(ctx context.Context) error {
	return e.rt.Close(ctx)
}

// RunFile loads a WASM module from path and calls exported run() -> i32.
func (e *Engine) RunFile(ctx context.Context, path string) (Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Result{}, err
	}
	return e.Run(ctx, data, path)
}

// Run instantiates WASM bytes and invokes run().
func (e *Engine) Run(ctx context.Context, wasm []byte, name string) (Result, error) {
	mod, err := e.rt.CompileModule(ctx, wasm)
	if err != nil {
		return Result{}, fmt.Errorf("compile wasm %s: %w", name, err)
	}
	inst, err := e.rt.InstantiateModule(ctx, mod, wazero.NewModuleConfig())
	if err != nil {
		return Result{}, fmt.Errorf("instantiate wasm %s: %w", name, err)
	}
	defer inst.Close(ctx)

	fn := inst.ExportedFunction(exportRun)
	if fn == nil {
		return Result{}, fmt.Errorf("wasm %s: missing export %q", name, exportRun)
	}
	ret, err := fn.Call(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("wasm %s: call %s: %w", name, exportRun, err)
	}
	if len(ret) != 1 {
		return Result{}, fmt.Errorf("wasm %s: %s returned %d values", name, exportRun, len(ret))
	}
	code := api.DecodeU32(ret[0])
	return Result{Code: code, Name: name}, nil
}

// RunAll executes each plugin path; non-zero run() code is a violation.
func RunAll(ctx context.Context, eng *Engine, paths []string) error {
	for _, p := range paths {
		res, err := eng.RunFile(ctx, p)
		if err != nil {
			return err
		}
		if res.Code != 0 {
			return fmt.Errorf("plugin %s: violation code %d", p, res.Code)
		}
	}
	return nil
}
