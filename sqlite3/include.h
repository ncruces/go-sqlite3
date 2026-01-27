#pragma once

#include <assert.h>
#include <stddef.h>

// https://github.com/JuliaLang/julia/blob/v1.9.4/src/julia.h#L67-L68
#define container_of(ptr, type, member) \
  ((type*)((char*)(ptr) - offsetof(type, member)))

typedef void* go_handle;
void go_destroy(go_handle);
static_assert(sizeof(go_handle) == 4, "Unexpected size");
