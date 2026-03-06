#pragma once

#include <stddef.h>

void* memset(void*, int, size_t);
void* memcpy(void* restrict, const void* restrict, size_t);
void* memmove(void*, const void*, size_t);

void* memchr(const void*, int, size_t);
void* memrchr(const void*, int, size_t);
int memcmp(const void*, const void*, size_t);

size_t strlen(const char*);
size_t strspn(const char*, const char*);
size_t strcspn(const char*, const char*);
char* strchr(const char*, int);
char* strrchr(const char*, int);
char* strchrnul(const char*, int);
int strcmp(const char*, const char*);
int strncmp(const char*, const char*, size_t);
