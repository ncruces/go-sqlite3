# Using SIMD for libc

I found that implementing some libc functions with Wasm SIMD128 can make them significantly faster.

Rough numbers for [wazero](https://wazero.io/):

  function   | speedup
------------ | -----
`strlen`     |  4.1×
`memchr`     |  4.1×
`strchr`     |  4.0×
`strrchr`    |  9.1×
`memcmp`     | 13.0×
`strcmp`     | 10.4×
`strncmp`    | 15.7×
`strcasecmp` |  8.8×
`strncasecmp`|  8.6×
`strspn`     |  9.9×
`strcspn`    |  9.0×
`memmem`     |  2.2×
`strstr`     |  5.5×
`strcasestr` | 25.2×

For functions where musl uses SWAR on a 4-byte `size_t`,
the improvement is around 4×.
This is very close to the expected theoretical improvement,
as we're processing 4× the bytes per cycle (16 _vs._ 4).

For other functions where there's no algorithmic change,
the improvement is around 8×.
These functions are harder to optimize
(which is why musl doesn't bother with SWAR),
so getting an 8× improvement from processing 16× bytes seems decent.

String search is harder to compare, since there are algorithmic changes,
and different needles produce very different numbers.
We use [Quick Search](https://igm.univ-mlv.fr/~lecroq/string/node19.html) for `memmem`,
and a [Rabin–Karp](https://igm.univ-mlv.fr/~lecroq/string/node5.html) for `strstr` and `strcasestr`;
musl uses [Two Way](https://igm.univ-mlv.fr/~lecroq/string/node26.html) for `memmem` and `strstr`,
and [brute force](https://igm.univ-mlv.fr/~lecroq/string/node3.html) for `strcasestr`.
Unlike Two-Way, both replacements can go quadratic for long, periodic needles.