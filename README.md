# Go OpenCC

Go wrapper for OpenCC (Open Chinese Convert) using WebAssembly, enabling Chinese text conversion between Simplified and Traditional Chinese.

## Features

- Convert between Simplified and Traditional Chinese
- Support for various regional variants (Taiwan, Hong Kong)
- WebAssembly-based implementation for cross-platform compatibility
- High performance with wazero runtime
- Thread-safe operations
- Memory efficient

## Installation

```bash
go get github.com/your-username/go-opencc
```

## Prerequisites

To build the WebAssembly module, you need:

1. **WASI SDK**: Download from [wasi-sdk releases](https://github.com/WebAssembly/wasi-sdk/releases)
2. **CMake**: Version 3.5 or higher
3. **wasm-opt** (optional): For optimization, install from [binaryen](https://github.com/WebAssembly/binaryen)

## Building

1. Clone the repository with submodules:

```bash
git clone --recursive https://github.com/your-username/go-opencc.git
cd go-opencc
```

2. Build the WebAssembly module:

```bash
./build.sh
```

3. Download Go dependencies:

```bash
go mod tidy
```

## Usage

### Simple Conversion Functions

```go
package main

import (
    "fmt"
    "log"

    "github.com/your-username/go-opencc"
)

func main() {
    // Convert Simplified to Traditional Chinese
    result, err := opencc.ConvertS2T("简体字")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(result) // Output: 簡體字

    // Convert Traditional to Simplified Chinese
    result, err = opencc.ConvertT2S("繁體字")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(result) // Output: 繁体字
}
```

### Using Converter Instance

For better performance when doing multiple conversions, use a converter instance:

```go
package main

import (
    "fmt"
    "log"

    "github.com/your-username/go-opencc"
)

func main() {
    // Create a converter instance
    converter, err := opencc.NewConverter("s2t.json")
    if err != nil {
        log.Fatal(err)
    }
    defer converter.Close()

    // Convert multiple texts
    texts := []string{"简体字", "测试", "转换"}
    for _, text := range texts {
        result, err := converter.Convert(text)
        if err != nil {
            log.Fatal(err)
        }
        fmt.Printf("%s -> %s\n", text, result)
    }
}
```

## API Reference

### Functions

#### `ConvertS2T(input string) (string, error)`

Converts Simplified Chinese to Traditional Chinese.

#### `ConvertT2S(input string) (string, error)`

Converts Traditional Chinese to Simplified Chinese.

#### `NewConverter(configFile string) (*Converter, error)`

Creates a new converter instance with the specified configuration file.

### Types

#### `type Converter struct`

Represents an OpenCC converter instance.

**Methods:**

- `Convert(input string) (string, error)` - Converts text using the converter
- `Close() error` - Closes the converter and releases resources

### Errors

- `ErrInvalidConverter` - Returned when converter creation fails
- `ErrConversionFailed` - Returned when text conversion fails

## Testing

Run tests:

```bash
go test
```

Run benchmarks:

```bash
go test -bench=.
```

## Performance

This implementation uses WebAssembly with the wazero runtime, providing:

- Fast startup times
- Low memory overhead
- Thread-safe operations
- Cross-platform compatibility

## License

This project is licensed under the Apache License 2.0 - see the OpenCC project for details.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Run tests and ensure they pass
6. Submit a pull request

## Acknowledgments

- [OpenCC](https://github.com/BYVoid/OpenCC) - The original OpenCC library
- [wazero](https://github.com/tetratelabs/wazero) - WebAssembly runtime for Go
