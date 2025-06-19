#!/bin/bash

# OpenCC WASM Build Script
# This script compiles OpenCC to WebAssembly using WASI SDK

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${GREEN}Building OpenCC WASM...${NC}"

# Check for required tools
if ! command -v cmake &> /dev/null; then
    echo -e "${RED}Error: cmake not found${NC}"
    exit 1
fi

if [ ! -d "/opt/wasi-sdk" ]; then
    echo -e "${RED}Error: WASI SDK not found at /opt/wasi-sdk${NC}"
    echo -e "${YELLOW}Please install WASI SDK from https://github.com/WebAssembly/wasi-sdk${NC}"
    exit 1
fi

# Create build directory
mkdir -p build
cd build

# Configure with CMake
echo -e "${GREEN}Configuring CMake...${NC}"
cmake \
    -DCMAKE_TOOLCHAIN_FILE="/opt/wasi-sdk/share/cmake/wasi-sdk.cmake" \
    -DCMAKE_BUILD_TYPE=Release \
    -DBUILD_SHARED_LIBS=OFF \
    -DENABLE_GTEST=OFF \
    -DENABLE_BENCHMARK=OFF \
    -DBUILD_TESTING=OFF \
    -DBUILD_DOCUMENTATION=OFF \
    -DBUILD_PYTHON=OFF \
    ..

# Build
echo -e "${GREEN}Building...${NC}"
cmake --build . --target opencc_wasm -j$(nproc)

# Move the WASM file to root directory
if [ -f "opencc.wasm" ]; then
    mv opencc.wasm ../
    echo -e "${GREEN}Build successful! Generated opencc.wasm${NC}"
    
    # Go back to root directory
    cd ..
    
    # Optimize with wasm-opt if available
    if command -v wasm-opt &> /dev/null; then
        echo -e "${GREEN}Optimizing with wasm-opt...${NC}"
        wasm-opt --strip -c -O3 opencc.wasm -o opencc.wasm
        echo -e "${GREEN}Optimization complete!${NC}"
    else
        echo -e "${YELLOW}wasm-opt not found, skipping optimization${NC}"
    fi
    
    # Organize data files
    echo -e "${BLUE}Organizing data files...${NC}"
    
    # Clean and recreate data directory
    rm -rf data
    mkdir -p data
    
    # Copy configuration files
    if [ -d "build" ]; then
        # Copy config files from build directory (generated during build)
        find build -name "*.json" -not -name "compile_commands.json" -not -name "config_test.json" -exec cp {} data/ \; 2>/dev/null || true
        
        # Copy dictionary files from build directory if they exist
        find build -name "*.ocd2" -exec cp {} data/ \; 2>/dev/null || true
    fi
    
    # If no dictionary files were found in build, copy from system
    if [ ! -f "data/STPhrases.ocd2" ] && [ -d "/usr/share/opencc" ]; then
        echo -e "${BLUE}Copying dictionary files from system installation...${NC}"
        cp /usr/share/opencc/*.ocd2 data/ 2>/dev/null || true
        # Also copy config files if not already present
        if [ ! -f "data/s2t.json" ]; then
            cp /usr/share/opencc/*.json data/ 2>/dev/null || true
        fi
    fi
    
    # If still no files, copy from source
    if [ ! -f "data/s2t.json" ] && [ -d "opencc/data/config" ]; then
        echo -e "${BLUE}Copying configuration files from source...${NC}"
        cp opencc/data/config/*.json data/ 2>/dev/null || true
    fi
    
    # Verify essential files exist
    ESSENTIAL_FILES=("s2t.json" "t2s.json" "STPhrases.ocd2" "STCharacters.ocd2")
    MISSING_FILES=()
    
    for file in "${ESSENTIAL_FILES[@]}"; do
        if [ ! -f "data/$file" ]; then
            MISSING_FILES+=("$file")
        fi
    done
    
    if [ ${#MISSING_FILES[@]} -gt 0 ]; then
        echo -e "${YELLOW}Warning: Some essential files are missing from data directory:${NC}"
        for file in "${MISSING_FILES[@]}"; do
            echo -e "${YELLOW}  - $file${NC}"
        done
        echo -e "${YELLOW}You may need to install OpenCC system package or build dictionaries manually${NC}"
    else
        echo -e "${GREEN}All essential data files are present${NC}"
    fi
    
    # Show data directory contents
    echo -e "${BLUE}Data directory contents:${NC}"
    ls -la data/ | head -10
    if [ $(ls data/ | wc -l) -gt 10 ]; then
        echo "... and $(( $(ls data/ | wc -l) - 10 )) more files"
    fi
    
    # Show file size
    SIZE=$(stat -c%s "opencc.wasm" 2>/dev/null || stat -f%z "opencc.wasm" 2>/dev/null || echo "unknown")
    echo -e "${GREEN}Generated opencc.wasm (${SIZE} bytes)${NC}"
    
    # Update .gitignore to include data directory
    if ! grep -q "^data/" .gitignore 2>/dev/null; then
        echo -e "${BLUE}Adding data/ to .gitignore...${NC}"
        echo "data/" >> .gitignore
    fi
    
    echo -e "${GREEN}Build complete! 🎉${NC}"
    echo -e "${BLUE}You can now run: go test -v${NC}"
    
else
    echo -e "${RED}Build failed: opencc.wasm not found${NC}"
    exit 1
fi 