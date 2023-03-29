package vfs

import (
	"context"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

func registerFunc1[T0, TR ~uint32](mod wazero.HostModuleBuilder, name string, fn func(ctx context.Context, mod api.Module, _ T0) TR) {
	mod.NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(
			func(ctx context.Context, mod api.Module, stack []uint64) {
				stack[0] = uint64(fn(ctx, mod, T0(stack[0])))
			}),
			[]api.ValueType{api.ValueTypeI32}, []api.ValueType{api.ValueTypeI32}).
		Export(name)
}

func registerFunc2[T0, T1, TR ~uint32](mod wazero.HostModuleBuilder, name string, fn func(ctx context.Context, mod api.Module, _ T0, _ T1) TR) {
	mod.NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(
			func(ctx context.Context, mod api.Module, stack []uint64) {
				stack[0] = uint64(fn(ctx, mod, T0(stack[0]), T1(stack[1])))
			}),
			[]api.ValueType{api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI32}).
		Export(name)
}

func registerFunc3[T0, T1, T2, TR ~uint32](mod wazero.HostModuleBuilder, name string, fn func(ctx context.Context, mod api.Module, _ T0, _ T1, _ T2) TR) {
	mod.NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(
			func(ctx context.Context, mod api.Module, stack []uint64) {
				stack[0] = uint64(fn(ctx, mod, T0(stack[0]), T1(stack[1]), T2(stack[2])))
			}),
			[]api.ValueType{api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI32}).
		Export(name)
}

func registerFunc4[T0, T1, T2, T3, TR ~uint32](mod wazero.HostModuleBuilder, name string, fn func(ctx context.Context, mod api.Module, _ T0, _ T1, _ T2, _ T3) TR) {
	mod.NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(
			func(ctx context.Context, mod api.Module, stack []uint64) {
				stack[0] = uint64(fn(ctx, mod, T0(stack[0]), T1(stack[1]), T2(stack[2]), T3(stack[3])))
			}),
			[]api.ValueType{api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI32}).
		Export(name)
}

func registerFunc5[T0, T1, T2, T3, T4, TR ~uint32](mod wazero.HostModuleBuilder, name string, fn func(ctx context.Context, mod api.Module, _ T0, _ T1, _ T2, _ T3, _ T4) TR) {
	mod.NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(
			func(ctx context.Context, mod api.Module, stack []uint64) {
				stack[0] = uint64(fn(ctx, mod, T0(stack[0]), T1(stack[1]), T2(stack[2]), T3(stack[3]), T4(stack[4])))
			}),
			[]api.ValueType{api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI32}).
		Export(name)
}

func registerFuncRW(mod wazero.HostModuleBuilder, name string, fn func(ctx context.Context, mod api.Module, _, _, _ uint32, _ int64) _ErrorCode) {
	mod.NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(
			func(ctx context.Context, mod api.Module, stack []uint64) {
				stack[0] = uint64(fn(ctx, mod, uint32(stack[0]), uint32(stack[1]), uint32(stack[2]), int64(stack[3])))
			}),
			[]api.ValueType{api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI64}, []api.ValueType{api.ValueTypeI32}).
		Export(name)
}

func registerFuncT(mod wazero.HostModuleBuilder, name string, fn func(ctx context.Context, mod api.Module, _ uint32, _ int64) _ErrorCode) {
	mod.NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(
			func(ctx context.Context, mod api.Module, stack []uint64) {
				stack[0] = uint64(fn(ctx, mod, uint32(stack[0]), int64(stack[1])))
			}),
			[]api.ValueType{api.ValueTypeI32, api.ValueTypeI64}, []api.ValueType{api.ValueTypeI32}).
		Export(name)
}
