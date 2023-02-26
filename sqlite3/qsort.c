#include <stddef.h>

void qsort_r(void *, size_t, size_t,
             int (*)(const void *, const void *, void *), void *);

typedef int (*cmpfun)(const void *, const void *);

static int wrapper_cmp(const void *v1, const void *v2, void *cmp) {
  return ((cmpfun)cmp)(v1, v2);
}

void qsort(void *base, size_t nel, size_t width, cmpfun cmp) {
  qsort_r(base, nel, width, wrapper_cmp, cmp);
}