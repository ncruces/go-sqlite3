# SQLite Libc 

This is a minimal libc that offers just enough
to compile SQLite for wasm32 with nostdlib.

The allocator is either Doug Lea's malloc or a bump allocator,
and math is provided by the host side.
