package opencc

import (
	"context"
	"embed"
	_ "embed"
	"fmt"
	"io/fs"
	"os"
	"sync"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

//go:generate ./build.sh

//go:embed opencc.wasm
var binary []byte // WASM blob

//go:embed data/*
var dataFS embed.FS

var ErrInvalidConverter = fmt.Errorf("invalid converter")
var ErrConversionFailed = fmt.Errorf("conversion failed")

// Converter represents an OpenCC converter instance
type Converter struct {
	mod    *module
	handle uint32
}

// NewConverter creates a new OpenCC converter with the specified configuration.
// Common configurations:
//   - "s2t.json" - Simplified to Traditional Chinese
//   - "t2s.json" - Traditional to Simplified Chinese
//   - "s2tw.json" - Simplified to Traditional Chinese (Taiwan)
//   - "s2hk.json" - Simplified to Traditional Chinese (Hong Kong)
//   - "t2tw.json" - Traditional to Traditional Chinese (Taiwan)
//   - "t2hk.json" - Traditional to Traditional Chinese (Hong Kong)
func NewConverter(configFile string) (*Converter, error) {
	mod, err := newModule()
	if err != nil {
		return nil, fmt.Errorf("init module: %w", err)
	}

	var handle uint32
	if err := mod.call("opencc_open", &handle, configFile); err != nil {
		mod.close()
		return nil, fmt.Errorf("open converter: %w", err)
	}

	if handle == ^uint32(0) { // (opencc_t)-1
		mod.close()
		return nil, ErrInvalidConverter
	}

	return &Converter{
		mod:    mod,
		handle: handle,
	}, nil
}

// Convert converts the input text using the converter
func (c *Converter) Convert(input string) (string, error) {
	if c.mod == nil || c.handle == ^uint32(0) {
		return "", ErrInvalidConverter
	}

	var result string
	if err := c.mod.call("opencc_convert", &result, c.handle, input); err != nil {
		return "", fmt.Errorf("convert: %w", err)
	}

	if result == "" {
		return "", ErrConversionFailed
	}

	return result, nil
}

// Close closes the converter and releases resources
func (c *Converter) Close() error {
	if c.mod == nil {
		return nil
	}

	if c.handle != ^uint32(0) {
		var result int32
		if err := c.mod.call("opencc_close", &result, c.handle); err != nil {
			// Log the error but continue with cleanup
			fmt.Printf("Warning: error closing OpenCC converter: %v\n", err)
		}
		c.handle = ^uint32(0)
	}

	c.mod.close()
	c.mod = nil
	return nil
}

// ConvertS2T converts Simplified Chinese to Traditional Chinese
func ConvertS2T(input string) (string, error) {
	mod, err := newModule()
	if err != nil {
		return "", fmt.Errorf("init module: %w", err)
	}
	defer mod.close()

	var result string
	if err := mod.call("opencc_s2t", &result, input); err != nil {
		return "", fmt.Errorf("convert: %w", err)
	}

	// Empty result is only an error if input was non-empty
	if result == "" && input != "" {
		return "", ErrConversionFailed
	}

	return result, nil
}

// ConvertT2S converts Traditional Chinese to Simplified Chinese
func ConvertT2S(input string) (string, error) {
	mod, err := newModule()
	if err != nil {
		return "", fmt.Errorf("init module: %w", err)
	}
	defer mod.close()

	var result string
	if err := mod.call("opencc_t2s", &result, input); err != nil {
		return "", fmt.Errorf("convert: %w", err)
	}

	// Empty result is only an error if input was non-empty
	if result == "" && input != "" {
		return "", ErrConversionFailed
	}

	return result, nil
}

// module wraps wazero module for OpenCC
type module struct {
	mod api.Module
}

var (
	rtMu sync.Mutex
	rt   wazero.Runtime
	cm   wazero.CompiledModule
)

func newModule() (*module, error) {
	rtMu.Lock()
	defer rtMu.Unlock()

	if rt == nil {
		rt = wazero.NewRuntime(context.Background())
		if _, err := wasi_snapshot_preview1.Instantiate(context.Background(), rt); err != nil {
			return nil, fmt.Errorf("instantiate wasi: %w", err)
		}

		// Create env module for C++ runtime functions
		envModuleBuilder := rt.NewHostModuleBuilder("env")

		// C++ exception handling functions
		envModuleBuilder.NewFunctionBuilder().WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			// __cxa_allocate_exception - allocate memory for exception
			size := uint32(stack[0])
			malloc := mod.ExportedFunction("malloc")
			ret, _ := malloc.Call(ctx, uint64(size))
			stack[0] = ret[0]
		}), []api.ValueType{api.ValueTypeI32}, []api.ValueType{api.ValueTypeI32}).Export("__cxa_allocate_exception")

		envModuleBuilder.NewFunctionBuilder().WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			// __cxa_throw - throw exception, try to get error info
			exceptionPtr := uint32(stack[0])
			fmt.Printf("C++ exception thrown - ptr: %v, type: %v, destructor: %v\n", stack[0], stack[1], stack[2])

			// Try to read error string from memory if possible
			mem := mod.Memory()
			if mem != nil && exceptionPtr > 0 {
				errorMsg := ""
				for i := uint32(0); i < 256; i++ { // Read max 256 bytes
					b, ok := mem.ReadByte(exceptionPtr + i)
					if !ok || b == 0 {
						break
					}
					if b >= 32 && b <= 126 { // Only printable ASCII
						errorMsg += string(b)
					}
				}
				if errorMsg != "" {
					fmt.Printf("Exception message: %s\n", errorMsg)
				}
			}

			panic("OpenCC error: failed to load or process configuration")
		}), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{}).Export("__cxa_throw")

		envModuleBuilder.NewFunctionBuilder().WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			// __cxa_free_exception - free exception memory
			ptr := uint32(stack[0])
			free := mod.ExportedFunction("free")
			if _, err := free.Call(ctx, uint64(ptr)); err != nil {
				fmt.Printf("Warning: error freeing exception memory: %v\n", err)
			}
		}), []api.ValueType{api.ValueTypeI32}, []api.ValueType{}).Export("__cxa_free_exception")

		// Personality function for exception handling
		envModuleBuilder.NewFunctionBuilder().WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			// Just return 0 to indicate we don't handle exceptions
			stack[0] = 0
		}), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI64, api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI32}).Export("__gxx_personality_v0")

		// Type info functions
		envModuleBuilder.NewFunctionBuilder().WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			// __cxa_begin_catch - begin catching exception
			// Return the exception pointer as-is (pass-through)
			// stack[0] already contains the input, no assignment needed
		}), []api.ValueType{api.ValueTypeI32}, []api.ValueType{api.ValueTypeI32}).Export("__cxa_begin_catch")

		envModuleBuilder.NewFunctionBuilder().WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			// __cxa_end_catch - end catching exception (no-op)
		}), []api.ValueType{}, []api.ValueType{}).Export("__cxa_end_catch")

		_, err := envModuleBuilder.Instantiate(context.Background())
		if err != nil {
			return nil, fmt.Errorf("instantiate env module: %w", err)
		}

		var err2 error
		cm, err2 = rt.CompileModule(context.Background(), binary)
		if err2 != nil {
			return nil, fmt.Errorf("compile module: %w", err2)
		}
	}

	// Configure module with embedded file system access
	// Create a sub-filesystem from the embedded data directory
	dataSubFS, err := fs.Sub(dataFS, "data")
	if err != nil {
		return nil, fmt.Errorf("create data sub-filesystem: %w", err)
	}

	config := wazero.NewModuleConfig().
		WithFS(dataSubFS). // Mount embedded data directory as root
		WithArgs("opencc").
		WithName("opencc").
		WithStdout(os.Stdout).
		WithStderr(os.Stderr)

	mod, err := rt.InstantiateModule(context.Background(), cm, config)
	if err != nil {
		return nil, fmt.Errorf("instantiate module: %w", err)
	}

	return &module{mod: mod}, nil
}

func (m *module) malloc(size uint32) uint32 {
	ret, _ := m.mod.ExportedFunction("malloc").Call(context.Background(), uint64(size))
	return uint32(ret[0])
}

func (m *module) call(name string, dest any, args ...any) error {
	fn := m.mod.ExportedFunction(name)
	if fn == nil {
		return fmt.Errorf("function %s not found", name)
	}

	var params []uint64
	var ptrsToFree []uint32

	defer func() {
		for _, ptr := range ptrsToFree {
			if ptr != 0 {
				if _, err := m.mod.ExportedFunction("free").Call(context.Background(), uint64(ptr)); err != nil {
					// Log error but don't fail since this is cleanup
					fmt.Printf("Warning: error freeing memory: %v\n", err)
				}
			}
		}
	}()

	for _, arg := range args {
		switch v := arg.(type) {
		case string:
			ptr := makeString(m, v)
			ptrsToFree = append(ptrsToFree, ptr)
			params = append(params, uint64(ptr))
		case uint32:
			params = append(params, uint64(v))
		case int32:
			params = append(params, uint64(uint32(v)))
		default:
			return fmt.Errorf("unsupported argument type: %T", arg)
		}
	}

	ret, err := fn.Call(context.Background(), params...)
	if err != nil {
		return fmt.Errorf("call %s: %w", name, err)
	}

	if len(ret) == 0 {
		return nil
	}

	switch d := dest.(type) {
	case *string:
		ptr := uint32(ret[0])
		if ptr == 0 {
			*d = ""
		} else {
			*d = readString(m, ptr)
			// Free the returned string
			if _, err := m.mod.ExportedFunction("opencc_convert_free").Call(context.Background(), uint64(ptr)); err != nil {
				fmt.Printf("Warning: error freeing converted string: %v\n", err)
			}
		}
	case *uint32:
		*d = uint32(ret[0])
	case *int32:
		*d = int32(ret[0])
	default:
		return fmt.Errorf("unsupported destination type: %T", dest)
	}

	return nil
}

func (m *module) close() {
	if m.mod != nil {
		m.mod.Close(context.Background())
	}
}

func makeString(m *module, s string) uint32 {
	size := uint32(len(s) + 1)
	ptr := m.malloc(size)
	if ptr == 0 {
		return 0
	}

	if !m.mod.Memory().Write(ptr, append([]byte(s), 0)) {
		return 0
	}

	return ptr
}

func readString(m *module, ptr uint32) string {
	if ptr == 0 {
		return ""
	}

	mem := m.mod.Memory()
	var result []byte
	for {
		b, ok := mem.ReadByte(ptr)
		if !ok || b == 0 {
			break
		}
		result = append(result, b)
		ptr++
	}

	return string(result)
}
