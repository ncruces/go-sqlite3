(module $libc.wasm
 (type $0 (func (param i32 i32 i32) (result i32)))
 (type $1 (func (param i32 i32) (result i32)))
 (type $2 (func (param i32) (result i32)))
 (memory $0 256)
 (data $0 (i32.const 1024) "\01")
 (export "memory" (memory $0))
 (export "memset" (func $memset))
 (export "memcpy" (func $memcpy))
 (export "memcmp" (func $memcmp))
 (export "memchr" (func $memchr))
 (export "strlen" (func $strlen))
 (export "strcmp" (func $strcmp))
 (export "strncmp" (func $strncmp))
 (export "strchrnul" (func $strchrnul))
 (export "strchr" (func $strchr))
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
 (func $memchr (param $0 i32) (param $1 i32) (param $2 i32) (result i32)
  (local $3 v128)
  (block $block2
   (block $block1
    (block $block
     (if
      (i32.ge_u
       (local.get $2)
       (i32.const 16)
      )
      (then
       (local.set $3
        (i8x16.splat
         (local.get $1)
        )
       )
       (loop $label
        (br_if $block
         (v128.any_true
          (i8x16.eq
           (v128.load align=1
            (local.get $0)
           )
           (local.get $3)
          )
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
    (local.set $1
     (i32.and
      (local.get $1)
      (i32.const 255)
     )
    )
    (loop $label1
     (br_if $block2
      (i32.eq
       (i32.load8_u
        (local.get $0)
       )
       (local.get $1)
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
   (local.set $0
    (i32.const 0)
   )
  )
  (local.get $0)
 )
 (func $strlen (param $0 i32) (result i32)
  (local $1 i32)
  (local $2 i32)
  (local $scratch i32)
  (block $block
   (br_if $block
    (i32.gt_u
     (local.tee $1
      (local.get $0)
     )
     (local.tee $2
      (i32.sub
       (i32.shl
        (memory.size)
        (i32.const 16)
       )
       (i32.const 16)
      )
     )
    )
   )
   (loop $label
    (br_if $block
     (i32.eqz
      (i8x16.all_true
       (v128.load align=1
        (local.get $1)
       )
      )
     )
    )
    (br_if $label
     (i32.le_u
      (local.tee $1
       (i32.add
        (local.get $1)
        (i32.const 16)
       )
      )
      (local.get $2)
     )
    )
   )
  )
  (local.set $0
   (i32.add
    (i32.xor
     (local.get $0)
     (i32.const -1)
    )
    (local.get $1)
   )
  )
  (loop $label1
   (local.set $0
    (i32.add
     (local.get $0)
     (i32.const 1)
    )
   )
   (br_if $label1
    (block (result i32)
     (local.set $scratch
      (i32.load8_u
       (local.get $1)
      )
     )
     (local.set $1
      (i32.add
       (local.get $1)
       (i32.const 1)
      )
     )
     (local.get $scratch)
    )
   )
  )
  (local.get $0)
 )
 (func $strcmp (param $0 i32) (param $1 i32) (result i32)
  (local $2 i32)
  (local $3 i32)
  (local $4 v128)
  (block $block
   (br_if $block
    (i32.lt_u
     (local.tee $2
      (i32.sub
       (i32.shl
        (memory.size)
        (i32.const 16)
       )
       (i32.const 16)
      )
     )
     (local.get $0)
    )
   )
   (br_if $block
    (i32.gt_u
     (local.get $1)
     (local.get $2)
    )
   )
   (loop $label
    (br_if $block
     (v128.any_true
      (v128.xor
       (v128.load align=1
        (local.get $1)
       )
       (local.tee $4
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
       (local.get $4)
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
    (br_if $block
     (i32.gt_u
      (local.tee $0
       (i32.add
        (local.get $0)
        (i32.const 16)
       )
      )
      (local.get $2)
     )
    )
    (br_if $label
     (i32.le_u
      (local.get $1)
      (local.get $2)
     )
    )
   )
  )
  (if
   (i32.eq
    (local.tee $2
     (i32.load8_u
      (local.get $0)
     )
    )
    (local.tee $3
     (i32.load8_u
      (local.get $1)
     )
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
    (loop $label1
     (if
      (i32.eqz
       (local.get $2)
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
   (local.get $2)
   (local.get $3)
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
 (func $strchrnul (param $0 i32) (param $1 i32) (result i32)
  (local $2 i32)
  (local $3 v128)
  (local $4 v128)
  (block $block
   (br_if $block
    (i32.lt_u
     (local.tee $2
      (i32.sub
       (i32.shl
        (memory.size)
        (i32.const 16)
       )
       (i32.const 16)
      )
     )
     (local.get $0)
    )
   )
   (local.set $3
    (i8x16.splat
     (local.get $1)
    )
   )
   (loop $label
    (br_if $block
     (i32.eqz
      (i8x16.all_true
       (local.tee $4
        (v128.load align=1
         (local.get $0)
        )
       )
      )
     )
    )
    (br_if $block
     (v128.any_true
      (i8x16.eq
       (local.get $4)
       (local.get $3)
      )
     )
    )
    (br_if $label
     (i32.le_u
      (local.tee $0
       (i32.add
        (local.get $0)
        (i32.const 16)
       )
      )
      (local.get $2)
     )
    )
   )
  )
  (local.set $1
   (i32.extend8_s
    (local.get $1)
   )
  )
  (local.set $0
   (i32.sub
    (local.get $0)
    (i32.const 1)
   )
  )
  (loop $label1
   (br_if $label1
    (select
     (local.tee $2
      (i32.load8_s
       (local.tee $0
        (i32.add
         (local.get $0)
         (i32.const 1)
        )
       )
      )
     )
     (i32.const 0)
     (i32.ne
      (local.get $1)
      (local.get $2)
     )
    )
   )
  )
  (local.get $0)
 )
 (func $strchr (param $0 i32) (param $1 i32) (result i32)
  (local $2 i32)
  (local $3 v128)
  (local $4 v128)
  (block $block
   (br_if $block
    (i32.lt_u
     (local.tee $2
      (i32.sub
       (i32.shl
        (memory.size)
        (i32.const 16)
       )
       (i32.const 16)
      )
     )
     (local.get $0)
    )
   )
   (local.set $3
    (i8x16.splat
     (local.get $1)
    )
   )
   (loop $label
    (br_if $block
     (i32.eqz
      (i8x16.all_true
       (local.tee $4
        (v128.load align=1
         (local.get $0)
        )
       )
      )
     )
    )
    (br_if $block
     (v128.any_true
      (i8x16.eq
       (local.get $4)
       (local.get $3)
      )
     )
    )
    (br_if $label
     (i32.le_u
      (local.tee $0
       (i32.add
        (local.get $0)
        (i32.const 16)
       )
      )
      (local.get $2)
     )
    )
   )
  )
  (local.set $1
   (i32.extend8_s
    (local.get $1)
   )
  )
  (local.set $0
   (i32.sub
    (local.get $0)
    (i32.const 1)
   )
  )
  (loop $label1
   (br_if $label1
    (select
     (local.tee $2
      (i32.load8_s
       (local.tee $0
        (i32.add
         (local.get $0)
         (i32.const 1)
        )
       )
      )
     )
     (i32.const 0)
     (i32.ne
      (local.get $1)
      (local.get $2)
     )
    )
   )
  )
  (select
   (local.get $0)
   (i32.const 0)
   (i32.eq
    (local.get $1)
    (local.get $2)
   )
  )
 )
 ;; features section: mutable-globals, nontrapping-float-to-int, simd, bulk-memory, sign-ext, reference-types, multivalue, bulk-memory-opt
)

