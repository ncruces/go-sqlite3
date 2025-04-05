(module $libc.wasm
 (type $0 (func (param i32 i32 i32) (result i32)))
 (type $1 (func (param i32 i32) (result i32)))
 (memory $0 256)
 (data $0 (i32.const 1024) "\01")
 (export "memory" (memory $0))
 (export "memset" (func $memset))
 (export "memcpy" (func $memcpy))
 (export "memcmp" (func $memcmp))
 (export "strcmp" (func $strcmp))
 (export "strncmp" (func $strncmp))
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
  (block $block1
   (block $block
    (if
     (i32.ge_u
      (local.get $2)
      (i32.const 16)
     )
     (then
      (loop $label
       (br_if $block
        (v128.any_true
         (v128.xor
          (v128.load align=1
           (local.get $1)
          )
          (v128.load align=1
           (local.get $0)
          )
         )
        )
       )
       (local.set $1
        (i32.add
         (local.get $1)
         (i32.const 16)
        )
       )
       (local.set $0
        (i32.add
         (local.get $0)
         (i32.const 16)
        )
       )
       (br_if $label
        (i32.gt_u
         (local.tee $2
          (i32.sub
           (local.get $2)
           (i32.const 16)
          )
         )
         (i32.const 15)
        )
       )
      )
     )
    )
    (br_if $block1
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
 (func $strcmp (param $0 i32) (param $1 i32) (result i32)
  (local $2 i32)
  (local $3 i32)
  (local $4 v128)
  (local $5 v128)
  (local.set $3
   (block $block (result i32)
    (if
     (i32.and
      (i32.or
       (local.get $0)
       (local.get $1)
      )
      (i32.const 15)
     )
     (then
      (local.set $2
       (i32.load8_u
        (local.get $0)
       )
      )
      (br $block
       (i32.load8_u
        (local.get $1)
       )
      )
     )
    )
    (if
     (v128.any_true
      (v128.xor
       (local.tee $5
        (v128.load
         (local.get $1)
        )
       )
       (local.tee $4
        (v128.load
         (local.get $0)
        )
       )
      )
     )
     (then
      (local.set $2
       (i8x16.extract_lane_u 0
        (local.get $4)
       )
      )
      (br $block
       (i8x16.extract_lane_u 0
        (local.get $5)
       )
      )
     )
    )
    (loop $label
     (if
      (i32.eqz
       (i8x16.all_true
        (local.get $4)
       )
      )
      (then
       (return
        (i32.const 0)
       )
      )
     )
     (local.set $4
      (v128.load offset=16
       (local.get $0)
      )
     )
     (local.set $5
      (v128.load offset=16
       (local.get $1)
      )
     )
     (local.set $1
      (i32.add
       (local.get $1)
       (i32.const 16)
      )
     )
     (local.set $0
      (i32.add
       (local.get $0)
       (i32.const 16)
      )
     )
     (br_if $label
      (i32.eqz
       (v128.any_true
        (v128.xor
         (local.get $5)
         (local.get $4)
        )
       )
      )
     )
    )
    (local.set $2
     (i8x16.extract_lane_u 0
      (local.get $4)
     )
    )
    (i8x16.extract_lane_u 0
     (local.get $5)
    )
   )
  )
  (if
   (i32.eq
    (i32.and
     (local.get $2)
     (i32.const 255)
    )
    (i32.and
     (local.get $3)
     (i32.const 255)
    )
   )
   (then
    (local.set $0
     (i32.add
      (local.get $0)
      (i32.const 1)
     )
    )
    (local.set $1
     (i32.add
      (local.get $1)
      (i32.const 1)
     )
    )
    (local.set $2
     (local.get $3)
    )
    (loop $label1
     (if
      (i32.eqz
       (i32.and
        (local.get $2)
        (i32.const 255)
       )
      )
      (then
       (return
        (i32.const 0)
       )
      )
     )
     (local.set $3
      (i32.load8_u
       (local.get $1)
      )
     )
     (local.set $2
      (i32.load8_u
       (local.get $0)
      )
     )
     (local.set $0
      (i32.add
       (local.get $0)
       (i32.const 1)
      )
     )
     (local.set $1
      (i32.add
       (local.get $1)
       (i32.const 1)
      )
     )
     (br_if $label1
      (i32.eq
       (local.get $2)
       (local.get $3)
      )
     )
    )
   )
  )
  (i32.sub
   (i32.and
    (local.get $2)
    (i32.const 255)
   )
   (i32.and
    (local.get $3)
    (i32.const 255)
   )
  )
 )
 (func $strncmp (param $0 i32) (param $1 i32) (param $2 i32) (result i32)
  (local $3 i32)
  (local $4 i32)
  (local $5 v128)
  (block $block
   (if
    (i32.ge_u
     (local.get $2)
     (i32.const 16)
    )
    (then
     (loop $label
      (br_if $block
       (v128.any_true
        (v128.xor
         (v128.load align=1
          (local.get $1)
         )
         (local.tee $5
          (v128.load align=1
           (local.get $0)
          )
         )
        )
       )
      )
      (if
       (i32.eqz
        (i8x16.all_true
         (local.get $5)
        )
       )
       (then
        (return
         (i32.const 0)
        )
       )
      )
      (local.set $1
       (i32.add
        (local.get $1)
        (i32.const 16)
       )
      )
      (local.set $0
       (i32.add
        (local.get $0)
        (i32.const 16)
       )
      )
      (br_if $label
       (i32.gt_u
        (local.tee $2
         (i32.sub
          (local.get $2)
          (i32.const 16)
         )
        )
        (i32.const 15)
       )
      )
     )
    )
   )
   (br_if $block
    (local.get $2)
   )
   (return
    (i32.const 0)
   )
  )
  (local.set $2
   (i32.sub
    (local.get $2)
    (i32.const 1)
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
   (if
    (local.get $3)
    (then
     (local.set $2
      (i32.sub
       (local.tee $3
        (local.get $2)
       )
       (i32.const 1)
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
      (local.get $3)
     )
    )
   )
  )
  (i32.const 0)
 )
 ;; features section: mutable-globals, nontrapping-float-to-int, simd, bulk-memory, sign-ext, reference-types, multivalue, bulk-memory-opt
)

