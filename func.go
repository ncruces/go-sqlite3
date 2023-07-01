package sqlite3

import (
	"context"

	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// CreateCollation defines a new collating sequence.
//
// https://www.sqlite.org/c3ref/create_collation.html
func (c *Conn) CreateCollation(name string, fn func(a, b []byte) int) error {
	namePtr := c.arena.string(name)
	funcPtr := util.AddHandle(c.ctx, fn)
	r := c.call(c.api.createCollation,
		uint64(c.handle), uint64(namePtr), uint64(funcPtr))
	if err := c.error(r); err != nil {
		util.DelHandle(c.ctx, funcPtr)
		return err
	}
	return nil
}

// CreateFunction defines a new scalar function.
//
// https://www.sqlite.org/c3ref/create_function.html
func (c *Conn) CreateFunction(name string, nArg int, flag FunctionFlag, fn func(ctx Context, arg ...Value)) error {
	namePtr := c.arena.string(name)
	funcPtr := util.AddHandle(c.ctx, fn)
	r := c.call(c.api.createFunction,
		uint64(c.handle), uint64(namePtr), uint64(nArg),
		uint64(flag), uint64(funcPtr))
	return c.error(r)
}

// CreateWindowFunction defines a new aggregate or window function.
//
// https://www.sqlite.org/c3ref/create_function.html
func (c *Conn) CreateWindowFunction(name string, nArg int, flag FunctionFlag, fn AggregateFunction) error {
	call := c.api.createAggregate
	namePtr := c.arena.string(name)
	funcPtr := util.AddHandle(c.ctx, fn)
	if _, ok := fn.(WindowFunction); ok {
		call = c.api.createWindow
	}
	r := c.call(call,
		uint64(c.handle), uint64(namePtr), uint64(nArg),
		uint64(flag), uint64(funcPtr))
	return c.error(r)
}

// ScalarFunction is the interface a scalar function should implement.
//
// https://www.sqlite.org/appfunc.html
type ScalarFunction interface {
	Func(ctx Context, arg ...Value)
}

// AggregateFunction is the interface an aggregate function should implement.
//
// https://www.sqlite.org/appfunc.html
type AggregateFunction interface {
	Step(ctx Context, arg ...Value)
	Final(ctx Context)
}

// WindowFunction is the interface an aggregate window function should implement.
//
// https://www.sqlite.org/windowfunctions.html
type WindowFunction interface {
	AggregateFunction
	Value(ctx Context)
	Inverse(ctx Context, arg ...Value)
}

func exportHostFunctions(env wazero.HostModuleBuilder) wazero.HostModuleBuilder {
	util.ExportFuncVI(env, "go_destroy", callbackDestroy)
	util.ExportFuncIIIIII(env, "go_compare", callbackCompare)
	util.ExportFuncVIII(env, "go_func", callbackFunc)
	util.ExportFuncVIII(env, "go_step", callbackStep)
	util.ExportFuncVI(env, "go_final", callbackFinal)
	util.ExportFuncVI(env, "go_value", callbackValue)
	util.ExportFuncVIII(env, "go_inverse", callbackInverse)
	return env
}

func callbackDestroy(ctx context.Context, mod api.Module, pApp uint32) {
	util.DelHandle(ctx, pApp)
}

func callbackCompare(ctx context.Context, mod api.Module, pApp, nKey1, pKey1, nKey2, pKey2 uint32) uint32 {
	fn := util.GetHandle(ctx, pApp).(func(a, b []byte) int)
	return uint32(fn(util.View(mod, pKey1, uint64(nKey1)), util.View(mod, pKey2, uint64(nKey2))))
}

func callbackFunc(ctx context.Context, mod api.Module, pCtx, nArg, pArg uint32) {
	module := ctx.Value(moduleKey{}).(*module)
	pApp := uint32(module.call(module.api.userData, uint64(pCtx)))
	fn := util.GetHandle(ctx, pApp).(func(ctx Context, arg ...Value))
	fn(Context{
		module: module,
		handle: pCtx,
	}, callbackArgs(module, nArg, pArg)...)
}

func callbackStep(ctx context.Context, mod api.Module, pCtx, nArg, pArg uint32) {
	module := ctx.Value(moduleKey{}).(*module)
	pApp := uint32(module.call(module.api.userData, uint64(pCtx)))
	fn := util.GetHandle(ctx, pApp).(AggregateFunction)
	fn.Step(Context{
		module: module,
		handle: pCtx,
	}, callbackArgs(module, nArg, pArg)...)
}

func callbackFinal(ctx context.Context, mod api.Module, pCtx uint32) {
	module := ctx.Value(moduleKey{}).(*module)
	pApp := uint32(module.call(module.api.userData, uint64(pCtx)))
	fn := util.GetHandle(ctx, pApp).(AggregateFunction)
	fn.Final(Context{
		module: module,
		handle: pCtx,
		final:  true,
	})
}

func callbackValue(ctx context.Context, mod api.Module, pCtx uint32) {
	module := ctx.Value(moduleKey{}).(*module)
	pApp := uint32(module.call(module.api.userData, uint64(pCtx)))
	fn := util.GetHandle(ctx, pApp).(WindowFunction)
	fn.Value(Context{
		module: module,
		handle: pCtx,
	})
}

func callbackInverse(ctx context.Context, mod api.Module, pCtx, nArg, pArg uint32) {
	module := ctx.Value(moduleKey{}).(*module)
	pApp := uint32(module.call(module.api.userData, uint64(pCtx)))
	fn := util.GetHandle(ctx, pApp).(WindowFunction)
	fn.Inverse(Context{
		module: module,
		handle: pCtx,
	}, callbackArgs(module, nArg, pArg)...)
}

func callbackArgs(module *module, nArg, pArg uint32) []Value {
	args := make([]Value, nArg)
	for i := range args {
		args[i] = Value{
			module: module,
			handle: util.ReadUint32(module.mod, pArg+ptrlen*uint32(i)),
		}
	}
	return args
}
