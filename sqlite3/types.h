#pragma once

typedef void *go_handle;

void go_destroy(go_handle);

static_assert(sizeof(go_handle) == 4, "Unexpected size");