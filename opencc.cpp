//go:build ignore
#include <cstdlib>
#include <cstring>
#include <iostream>

#include "opencc.h"

__attribute__((export_name("malloc"))) void *exported_malloc(size_t size) {
  return malloc(size);
}

__attribute__((export_name("free"))) void exported_free(void *ptr) {
  free(ptr);
}

__attribute__((export_name("opencc_open"))) opencc_t
opencc_wrapper_open(const char *config_file) {
  if (!config_file) {
    return opencc_open(OPENCC_DEFAULT_CONFIG_SIMP_TO_TRAD);
  }
  return opencc_open(config_file);
}

__attribute__((export_name("opencc_close"))) int
opencc_wrapper_close(opencc_t opencc) {
  return opencc_close(opencc);
}

__attribute__((export_name("opencc_convert"))) char *
opencc_wrapper_convert(opencc_t opencc, const char *input) {
  if (!opencc || !input) {
    return nullptr;
  }

  return opencc_convert_utf8(opencc, input, (size_t)-1);
}

__attribute__((export_name("opencc_convert_free"))) void
opencc_wrapper_convert_free(char *str) {
  opencc_convert_utf8_free(str);
}

__attribute__((export_name("opencc_error"))) const char *
opencc_wrapper_error() {
  return opencc_error();
}

// Convenience functions for common conversions
__attribute__((export_name("opencc_s2t"))) char *
opencc_s2t_convert(const char *input) {
  if (!input) {
    return nullptr;
  }

  opencc_t cc = opencc_open("s2t.json");
  if (cc == (opencc_t)-1) {
    return nullptr;
  }

  char *result = opencc_convert_utf8(cc, input, (size_t)-1);
  opencc_close(cc);
  return result;
}

__attribute__((export_name("opencc_t2s"))) char *
opencc_t2s_convert(const char *input) {
  if (!input) {
    return nullptr;
  }

  opencc_t cc = opencc_open("t2s.json");
  if (cc == (opencc_t)-1) {
    return nullptr;
  }

  char *result = opencc_convert_utf8(cc, input, (size_t)-1);
  opencc_close(cc);
  return result;
}