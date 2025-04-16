(module $libc.wasm
 (type $0 (func (param i32 i32 i32) (result i32)))
 (type $1 (func (param i32 i32) (result i32)))
 (type $2 (func (param i32) (result i32)))
 (memory $0 256)
 (data $0 (i32.const 65536) "\01")
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
 (export "strspn" (func $strspn))
 (export "strcspn" (func $strcspn))
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
  (local $3 i32)
  (local $4 i32)
  (local $5 i32)
  (local $6 v128)
  (local $7 v128)
  (local.set $4
   (i32.and
    (local.get $0)
    (i32.const 15)
   )
  )
  (block $block1
   (block $block
    (if
     (v128.any_true
      (local.tee $6
       (i8x16.eq
        (v128.load
         (local.tee $3
          (i32.and
           (local.get $0)
           (i32.const -16)
          )
         )
        )
        (local.tee $7
         (i8x16.splat
          (local.get $1)
         )
        )
       )
      )
     )
     (then
      (br_if $block
       (local.tee $1
        (i32.and
         (i8x16.bitmask
          (local.get $6)
         )
         (i32.shl
          (i32.const -1)
          (local.get $4)
         )
        )
       )
      )
     )
    )
    (br_if $block1
     (i32.gt_u
      (local.tee $1
       (i32.sub
        (i32.add
         (local.get $2)
         (local.get $4)
        )
        (i32.const 16)
       )
      )
      (local.get $2)
     )
    )
    (local.set $3
     (i32.add
      (i32.sub
       (local.get $0)
       (local.get $4)
      )
      (i32.const 16)
     )
    )
    (block $block2
     (loop $label
      (br_if $block2
       (v128.any_true
        (local.tee $6
         (i8x16.eq
          (v128.load
           (local.get $3)
          )
          (local.get $7)
         )
        )
       )
      )
      (local.set $3
       (i32.add
        (local.get $3)
        (i32.const 16)
       )
      )
      (br_if $label
       (i32.ge_u
        (local.get $1)
        (local.tee $1
         (i32.sub
          (local.get $1)
          (i32.const 16)
         )
        )
       )
      )
     )
     (br $block1)
    )
    (local.set $1
     (i8x16.bitmask
      (local.get $6)
     )
    )
   )
   (local.set $5
    (i32.add
     (local.get $3)
     (i32.ctz
      (local.get $1)
     )
    )
   )
  )
  (local.get $5)
 )
 (func $strlen (param $0 i32) (result i32)
  (local $1 i32)
  (local $2 i32)
  (local $3 v128)
  (block $block1
   (block $block
    (br_if $block
     (i8x16.all_true
      (local.tee $3
       (v128.load
        (local.tee $1
         (i32.and
          (local.get $0)
          (i32.const -16)
         )
        )
       )
      )
     )
    )
    (br_if $block
     (i32.eqz
      (local.tee $2
       (i32.and
        (i8x16.bitmask
         (i8x16.eq
          (local.get $3)
          (v128.const i32x4 0x00000000 0x00000000 0x00000000 0x00000000)
         )
        )
        (i32.shl
         (i32.const -1)
         (i32.and
          (local.get $0)
          (i32.const 15)
         )
        )
       )
      )
     )
    )
    (br $block1)
   )
   (loop $label
    (local.set $3
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
    (br_if $label
     (i8x16.all_true
      (local.get $3)
     )
    )
   )
   (local.set $2
    (i8x16.bitmask
     (i8x16.eq
      (local.get $3)
      (v128.const i32x4 0x00000000 0x00000000 0x00000000 0x00000000)
     )
    )
   )
  )
  (i32.add
   (i32.ctz
    (local.get $2)
   )
   (i32.sub
    (local.get $1)
    (local.get $0)
   )
  )
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
  (block $block1
   (block $block
    (br_if $block
     (i32.lt_u
      (local.tee $3
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
    (loop $label
     (br_if $block
      (i32.gt_u
       (local.get $1)
       (local.get $3)
      )
     )
     (br_if $block
      (i32.lt_u
       (local.get $2)
       (i32.const 16)
      )
     )
     (br_if $block1
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
     (local.set $2
      (i32.sub
       (local.get $2)
       (i32.const 16)
      )
     )
     (local.set $1
      (i32.add
       (local.get $1)
       (i32.const 16)
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
       (local.get $3)
      )
     )
    )
   )
   (br_if $block1
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
  (local $2 v128)
  (local $3 v128)
  (local $4 i32)
  (block $block
   (if
    (v128.any_true
     (local.tee $2
      (v128.or
       (i8x16.eq
        (local.tee $2
         (v128.load
          (local.tee $4
           (i32.and
            (local.get $0)
            (i32.const -16)
           )
          )
         )
        )
        (v128.const i32x4 0x00000000 0x00000000 0x00000000 0x00000000)
       )
       (i8x16.eq
        (local.get $2)
        (local.tee $3
         (i8x16.splat
          (local.get $1)
         )
        )
       )
      )
     )
    )
    (then
     (br_if $block
      (local.tee $0
       (i32.and
        (i8x16.bitmask
         (local.get $2)
        )
        (i32.shl
         (i32.const -1)
         (i32.and
          (local.get $0)
          (i32.const 15)
         )
        )
       )
      )
     )
    )
   )
   (loop $label
    (local.set $2
     (v128.load offset=16
      (local.get $4)
     )
    )
    (local.set $4
     (i32.add
      (local.get $4)
      (i32.const 16)
     )
    )
    (br_if $label
     (i32.eqz
      (v128.any_true
       (local.tee $2
        (v128.or
         (i8x16.eq
          (local.get $2)
          (v128.const i32x4 0x00000000 0x00000000 0x00000000 0x00000000)
         )
         (i8x16.eq
          (local.get $2)
          (local.get $3)
         )
        )
       )
      )
     )
    )
   )
   (local.set $0
    (i8x16.bitmask
     (local.get $2)
    )
   )
  )
  (i32.add
   (local.get $4)
   (i32.ctz
    (local.get $0)
   )
  )
 )
 (func $strchr (param $0 i32) (param $1 i32) (result i32)
  (local $2 v128)
  (local $3 v128)
  (local $4 i32)
  (block $block
   (if
    (v128.any_true
     (local.tee $2
      (v128.or
       (i8x16.eq
        (local.tee $2
         (v128.load
          (local.tee $4
           (i32.and
            (local.get $0)
            (i32.const -16)
           )
          )
         )
        )
        (v128.const i32x4 0x00000000 0x00000000 0x00000000 0x00000000)
       )
       (i8x16.eq
        (local.get $2)
        (local.tee $3
         (i8x16.splat
          (local.get $1)
         )
        )
       )
      )
     )
    )
    (then
     (br_if $block
      (local.tee $0
       (i32.and
        (i8x16.bitmask
         (local.get $2)
        )
        (i32.shl
         (i32.const -1)
         (i32.and
          (local.get $0)
          (i32.const 15)
         )
        )
       )
      )
     )
    )
   )
   (loop $label
    (local.set $2
     (v128.load offset=16
      (local.get $4)
     )
    )
    (local.set $4
     (i32.add
      (local.get $4)
      (i32.const 16)
     )
    )
    (br_if $label
     (i32.eqz
      (v128.any_true
       (local.tee $2
        (v128.or
         (i8x16.eq
          (local.get $2)
          (v128.const i32x4 0x00000000 0x00000000 0x00000000 0x00000000)
         )
         (i8x16.eq
          (local.get $2)
          (local.get $3)
         )
        )
       )
      )
     )
    )
   )
   (local.set $0
    (i8x16.bitmask
     (local.get $2)
    )
   )
  )
  (select
   (local.tee $0
    (i32.add
     (local.get $4)
     (i32.ctz
      (local.get $0)
     )
    )
   )
   (i32.const 0)
   (i32.eq
    (i32.load8_u
     (local.get $0)
    )
    (i32.and
     (local.get $1)
     (i32.const 255)
    )
   )
  )
 )
 (func $strspn (param $0 i32) (param $1 i32) (result i32)
  (local $2 i32)
  (local $3 i32)
  (if
   (i32.eqz
    (local.tee $2
     (i32.load8_u
      (local.get $1)
     )
    )
   )
   (then
    (return
     (i32.const 0)
    )
   )
  )
  (v128.store
   (i32.const 65520)
   (v128.const i32x4 0x00000000 0x00000000 0x00000000 0x00000000)
  )
  (v128.store
   (i32.const 65504)
   (v128.const i32x4 0x00000000 0x00000000 0x00000000 0x00000000)
  )
  (local.set $1
   (i32.add
    (local.get $1)
    (i32.const 1)
   )
  )
  (loop $label
   (i32.store
    (local.tee $3
     (i32.add
      (i32.and
       (i32.shr_u
        (local.get $2)
        (i32.const 3)
       )
       (i32.const 28)
      )
      (i32.const 65504)
     )
    )
    (i32.or
     (i32.load
      (local.get $3)
     )
     (i32.shl
      (i32.const 1)
      (local.get $2)
     )
    )
   )
   (local.set $2
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
   (br_if $label
    (local.get $2)
   )
  )
  (if
   (local.tee $2
    (i32.load8_u
     (local.tee $1
      (local.get $0)
     )
    )
   )
   (then
    (loop $label1
     (if
      (i32.eqz
       (i32.and
        (i32.shr_u
         (i32.load
          (i32.add
           (i32.and
            (i32.shr_u
             (local.get $2)
             (i32.const 3)
            )
            (i32.const 28)
           )
           (i32.const 65504)
          )
         )
         (local.get $2)
        )
        (i32.const 1)
       )
      )
      (then
       (return
        (i32.sub
         (local.get $1)
         (local.get $0)
        )
       )
      )
     )
     (local.set $2
      (i32.load8_u offset=1
       (local.get $1)
      )
     )
     (local.set $1
      (i32.add
       (local.get $1)
       (i32.const 1)
      )
     )
     (br_if $label1
      (local.get $2)
     )
    )
   )
  )
  (i32.sub
   (local.get $1)
   (local.get $0)
  )
 )
 (func $strcspn (param $0 i32) (param $1 i32) (result i32)
  (local $2 i32)
  (local $3 i32)
  (local $4 v128)
  (local $5 v128)
  (block $block
   (if
    (local.tee $2
     (i32.load8_u
      (local.get $1)
     )
    )
    (then
     (br_if $block
      (i32.load8_u offset=1
       (local.get $1)
      )
     )
    )
   )
   (block $block1
    (if
     (v128.any_true
      (local.tee $4
       (v128.or
        (i8x16.eq
         (local.tee $4
          (v128.load
           (local.tee $1
            (i32.and
             (local.get $0)
             (i32.const -16)
            )
           )
          )
         )
         (v128.const i32x4 0x00000000 0x00000000 0x00000000 0x00000000)
        )
        (i8x16.eq
         (local.get $4)
         (local.tee $5
          (i8x16.splat
           (local.get $2)
          )
         )
        )
       )
      )
     )
     (then
      (br_if $block1
       (local.tee $2
        (i32.and
         (i8x16.bitmask
          (local.get $4)
         )
         (i32.shl
          (i32.const -1)
          (i32.and
           (local.get $0)
           (i32.const 15)
          )
         )
        )
       )
      )
     )
    )
    (loop $label
     (local.set $4
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
     (br_if $label
      (i32.eqz
       (v128.any_true
        (local.tee $4
         (v128.or
          (i8x16.eq
           (local.get $4)
           (v128.const i32x4 0x00000000 0x00000000 0x00000000 0x00000000)
          )
          (i8x16.eq
           (local.get $4)
           (local.get $5)
          )
         )
        )
       )
      )
     )
    )
    (local.set $2
     (i8x16.bitmask
      (local.get $4)
     )
    )
   )
   (return
    (i32.sub
     (i32.add
      (local.get $1)
      (i32.ctz
       (local.get $2)
      )
     )
     (local.get $0)
    )
   )
  )
  (v128.store
   (i32.const 65520)
   (v128.const i32x4 0x00000000 0x00000000 0x00000000 0x00000000)
  )
  (v128.store
   (i32.const 65504)
   (v128.const i32x4 0x00000000 0x00000000 0x00000000 0x00000000)
  )
  (local.set $1
   (i32.add
    (local.get $1)
    (i32.const 1)
   )
  )
  (loop $label1
   (i32.store
    (local.tee $3
     (i32.add
      (i32.and
       (i32.shr_u
        (local.get $2)
        (i32.const 3)
       )
       (i32.const 28)
      )
      (i32.const 65504)
     )
    )
    (i32.or
     (i32.load
      (local.get $3)
     )
     (i32.shl
      (i32.const 1)
      (local.get $2)
     )
    )
   )
   (local.set $2
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
   (br_if $label1
    (local.get $2)
   )
  )
  (if
   (local.tee $2
    (i32.load8_u
     (local.tee $1
      (local.get $0)
     )
    )
   )
   (then
    (loop $label2
     (if
      (i32.and
       (i32.shr_u
        (i32.load
         (i32.add
          (i32.and
           (i32.shr_u
            (local.get $2)
            (i32.const 3)
           )
           (i32.const 28)
          )
          (i32.const 65504)
         )
        )
        (local.get $2)
       )
       (i32.const 1)
      )
      (then
       (return
        (i32.sub
         (local.get $1)
         (local.get $0)
        )
       )
      )
     )
     (local.set $2
      (i32.load8_u offset=1
       (local.get $1)
      )
     )
     (local.set $1
      (i32.add
       (local.get $1)
       (i32.const 1)
      )
     )
     (br_if $label2
      (local.get $2)
     )
    )
   )
  )
  (i32.sub
   (local.get $1)
   (local.get $0)
  )
 )
 ;; features section: mutable-globals, nontrapping-float-to-int, simd, bulk-memory, sign-ext, reference-types, multivalue, bulk-memory-opt
)

