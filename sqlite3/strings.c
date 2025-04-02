#include <limits.h>
#include <stddef.h>
#include <stdint.h>

#if defined(__wasm_bulk_memory__)
#define memset __builtin_memset
#define memcpy __builtin_memcpy
#define memmove __builtin_memmove
#endif

#define ONES (~(uintmax_t)(0) / UCHAR_MAX)
#define HIGHS (ONES * (UCHAR_MAX / 2 + 1))
#define HASZERO(x) ((x) - (typeof(x))(ONES) & ~(x) & (typeof(x))(HIGHS))
#define UNALIGNED(x) ((uintptr_t)(x) & (sizeof(*x) - 1))

int memcmp(const void *v1, const void *v2, size_t n) {
  typedef uint64_t __attribute__((__may_alias__)) word;

  const word *w1 = v1;
  const word *w2 = v2;
  if (!(UNALIGNED(w1) | UNALIGNED(w2))) {
    while (n >= sizeof(word) && *w1 == *w2) {
      n -= sizeof(word);
      w1++;
      w2++;
    }
  }

  const unsigned char *u1 = (const void *)w1;
  const unsigned char *u2 = (const void *)w2;
  while (n--) {
    if (*u1 != *u2) return *u1 - *u2;
    u1++;
    u2++;
  }
  return 0;
}

int strcmp(const char *c1, const char *c2) {
  typedef uintptr_t __attribute__((__may_alias__)) word;

  const word *w1 = (const void *)c1;
  const word *w2 = (const void *)c2;
  if (!(UNALIGNED(w1) | UNALIGNED(w2))) {
    while (*w1 == *w2) {
      if (HASZERO(*w1)) return 0;
      w1++;
      w2++;
    }
    c1 = (const void *)w1;
    c2 = (const void *)w2;
  }

  while (*c1 == *c2 && *c1) {
    c1++;
    c2++;
  }
  return *(unsigned char *)c1 - *(unsigned char *)c2;
}

int strncmp(const char *c1, const char *c2, size_t n) {
  typedef uintptr_t __attribute__((__may_alias__)) word;

  const word *w1 = (const void *)c1;
  const word *w2 = (const void *)c2;
  if (!(UNALIGNED(w1) | UNALIGNED(w2))) {
    while (n >= sizeof(word) && *w1 == *w2) {
      if ((n -= sizeof(word)) == 0 || HASZERO(*w1)) return 0;
      w1++;
      w2++;
    }
    c1 = (const void *)w1;
    c2 = (const void *)w2;
  }

  while (n-- && *c1 == *c2) {
    if (n == 0 || *c1 == 0) return 0;
    c1++;
    c2++;
  }
  return *(unsigned char *)c1 - *(unsigned char *)c2;
}

#undef UNALIGNED
#undef HASZERO
#undef HIGHS
#undef ONES