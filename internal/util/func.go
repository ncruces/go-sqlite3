package util

import (
	"context"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

type i32 interface{ ~int32 | ~uint32 }
type i64 interface{ ~int64 | ~uint64 }

func RegisterFuncII[TR, T0 i32](mod wazero.HostModuleBuilder, name string, fn func(ctx context.Context, mod api.Module, _ T0) TR) {
	mod.NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(
			func(ctx context.Context, mod api.Module, stack []uint64) {
				stack[0] = uint64(fn(ctx, mod, T0(stack[0])))
			}),
			[]api.ValueType{api.ValueTypeI32}, []api.ValueType{api.ValueTypeI32}).
		Export(name)
}

func RegisterFuncIII[TR, T0, T1 i32](mod wazero.HostModuleBuilder, name string, fn func(ctx context.Context, mod api.Module, _ T0, _ T1) TR) {
	mod.NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(
			func(ctx context.Context, mod api.Module, stack []uint64) {
				stack[0] = uint64(fn(ctx, mod, T0(stack[0]), T1(stack[1])))
			}),
			[]api.ValueType{api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI32}).
		Export(name)
}

func RegisterFuncIIII[TR, T0, T1, T2 i32](mod wazero.HostModuleBuilder, name string, fn func(ctx context.Context, mod api.Module, _ T0, _ T1, _ T2) TR) {
	mod.NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(
			func(ctx context.Context, mod api.Module, stack []uint64) {
				stack[0] = uint64(fn(ctx, mod, T0(stack[0]), T1(stack[1]), T2(stack[2])))
			}),
			[]api.ValueType{api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI32}).
		Export(name)
}

func RegisterFuncIIIII[TR, T0, T1, T2, T3 i32](mod wazero.HostModuleBuilder, name string, fn func(ctx context.Context, mod api.Module, _ T0, _ T1, _ T2, _ T3) TR) {
	mod.NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(
			func(ctx context.Context, mod api.Module, stack []uint64) {
				stack[0] = uint64(fn(ctx, mod, T0(stack[0]), T1(stack[1]), T2(stack[2]), T3(stack[3])))
			}),
			[]api.ValueType{api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI32}).
		Export(name)
}

func RegisterFuncIIIIII[TR, T0, T1, T2, T3, T4 i32](mod wazero.HostModuleBuilder, name string, fn func(ctx context.Context, mod api.Module, _ T0, _ T1, _ T2, _ T3, _ T4) TR) {
	mod.NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(
			func(ctx context.Context, mod api.Module, stack []uint64) {
				stack[0] = uint64(fn(ctx, mod, T0(stack[0]), T1(stack[1]), T2(stack[2]), T3(stack[3]), T4(stack[4])))
			}),
			[]api.ValueType{api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI32}).
		Export(name)
}

func RegisterFuncIIIIJ[TR, T0, T1, T2 i32, T3 i64](mod wazero.HostModuleBuilder, name string, fn func(ctx context.Context, mod api.Module, _ T0, _ T1, _ T2, _ T3) TR) {
	mod.NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(
			func(ctx context.Context, mod api.Module, stack []uint64) {
				stack[0] = uint64(fn(ctx, mod, T0(stack[0]), T1(stack[1]), T2(stack[2]), T3(stack[3])))
			}),
			[]api.ValueType{api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI64}, []api.ValueType{api.ValueTypeI32}).
		Export(name)
}

func RegisterFuncIIJ[TR, T0 i32, T1 i64](mod wazero.HostModuleBuilder, name string, fn func(ctx context.Context, mod api.Module, _ T0, _ T1) TR) {
	mod.NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(
			func(ctx context.Context, mod api.Module, stack []uint64) {
				stack[0] = uint64(fn(ctx, mod, T0(stack[0]), T1(stack[1])))
			}),
			[]api.ValueType{api.ValueTypeI32, api.ValueTypeI64}, []api.ValueType{api.ValueTypeI32}).
		Export(name)
}
