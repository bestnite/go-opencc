cmake_minimum_required(VERSION 3.5.0 FATAL_ERROR)
project(opencc_wasm)

set(CMAKE_BUILD_TYPE Release)
set(CMAKE_EXPORT_COMPILE_COMMANDS ON)
set(BUILD_SHARED_LIBS OFF)
set(ENABLE_GTEST OFF)
set(ENABLE_BENCHMARK OFF)
set(BUILD_TESTING OFF)
set(BUILD_DOCUMENTATION OFF)
set(BUILD_PYTHON OFF)
set(BUILD_TOOLS OFF)  # Disable tools for WASM build

# Enable CMP0079 to allow modifying targets from different directories
cmake_policy(SET CMP0079 NEW)

# Add global compiler definitions for WASI mmap emulation
add_definitions(-D_WASI_EMULATED_MMAN)

# Add opencc subdirectory
add_subdirectory(opencc)

# Include opencc headers
include_directories(
  opencc/src
  ${PROJECT_BINARY_DIR}/opencc/src
)

# Create the opencc wasm executable
add_executable(opencc_wasm opencc.cpp)
set_target_properties(opencc_wasm PROPERTIES SUFFIX ".wasm" OUTPUT_NAME "opencc")
target_compile_options(opencc_wasm PRIVATE 
  --target=wasm32-wasi 
  -g0 
  -O2 
  -D_WASI_EMULATED_MMAN
)
target_link_options(opencc_wasm PRIVATE 
  -Wl,--allow-undefined 
  -mexec-model=reactor
  -Wl,--export-dynamic
  -lwasi-emulated-mman
)
target_link_libraries(opencc_wasm PRIVATE libopencc)

# Copy data files to build directory root
add_custom_command(TARGET opencc_wasm POST_BUILD
  COMMAND ${CMAKE_COMMAND} -E copy_directory
  ${CMAKE_SOURCE_DIR}/opencc/data/config
  ${CMAKE_BINARY_DIR}
  COMMENT "Copying opencc config files to root"
)

 