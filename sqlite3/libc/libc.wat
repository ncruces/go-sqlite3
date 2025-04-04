(module $libc.wasm
 (type $0 (func (param i32 i32 i32) (result i32)))
 (memory $0 256)
 (data $0 (i32.const 1024) "\01")
 (export "memory" (memory $0))
 (export "memset" (func $memset))
 (export "memcpy" (func $memcpy))
 (export "memcmp" (func $memcmp))
 (func $memset (param $0 i32) (param $1 i32) (param $2 i32) (result i32)
  (memory.fill
   (local.get $0)
   (local.get $1)
   (local.get $2)
  )
  (local.get $0)
 )
 (func $memcpy (param $0 i32) (param $1 i32) (param $2 i32) (result i32)
  (memory.copy
   (local.get $0)
   (local.get $1)
   (local.get $2)
  )
  (local.get $0)
 )
 (func $memcmp (param $0 i32) (param $1 i32) (param $2 i32) (result i32)
  (local $3 i32)
  (local $4 i32)
  (block $block2
   (block $block1
    (block $block
     (br_if $block
      (i32.lt_u
       (local.get $2)
       (i32.const 8)
      )
     )
     (br_if $block
      (i32.and
       (i32.or
        (local.get $0)
        (local.get $1)
       )
       (i32.const 7)
      )
     )
     (loop $label
      (br_if $block1
       (i64.ne
        (i64.load
         (local.get $0)
        )
        (i64.load
         (local.get $1)
        )
       )
      )
      (local.set $1
       (i32.add
        (local.get $1)
        (i32.const 8)
       )
      )
      (local.set $0
       (i32.add
        (local.get $0)
        (i32.const 8)
       )
      )
      (br_if $label
       (i32.gt_u
        (local.tee $2
         (i32.sub
          (local.get $2)
          (i32.const 8)
         )
        )
        (i32.const 7)
       )
      )
     )
    )
    (br_if $block2
     (i32.eqz
      (local.get $2)
     )
    )
   )
   (loop $label1
    (if
     (i32.ne
      (local.tee $3
       (i32.load8_u
        (local.get $0)
       )
      )
      (local.tee $4
       (i32.load8_u
        (local.get $1)
       )
      )
     )
     (then
      (return
       (i32.sub
        (local.get $3)
        (local.get $4)
       )
      )
     )
    )
    (local.set $1
     (i32.add
      (local.get $1)
      (i32.const 1)
     )
    )
    (local.set $0
     (i32.add
      (local.get $0)
      (i32.const 1)
     )
    )
    (br_if $label1
     (local.tee $2
      (i32.sub
       (local.get $2)
       (i32.const 1)
      )
     )
    )
   )
  )
  (i32.const 0)
 )
 ;; features section: mutable-globals, nontrapping-float-to-int, simd, bulk-memory, sign-ext, reference-types, multivalue, bulk-memory-opt
)

