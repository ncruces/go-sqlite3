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

func exportHostFunctions(env wazero.HostModuleBuilder) wazero.HostModuleBuilder {
	util.ExportFuncVI(env, "go_destroy", cbDestroy)
	util.ExportFuncIIIIII(env, "go_compare", cbCompare)
	util.ExportFuncVIII(env, "go_func", cbFunc)
	util.ExportFuncVIII(env, "go_step", cbStep)
	util.ExportFuncVI(env, "go_final", cbFinal)
	util.ExportFuncVI(env, "go_value", cbValue)
	util.ExportFuncVIII(env, "go_inverse", cbInverse)
	return env
}

func cbDestroy(ctx context.Context, mod api.Module, pApp uint32) {
	util.DelHandle(ctx, pApp)
}

func cbCompare(ctx context.Context, mod api.Module, pApp, nKey1, pKey1, nKey2, pKey2 uint32) uint32 {
	fn := util.GetHandle(ctx, pApp).(func(a, b []byte) int)
	return uint32(fn(util.View(mod, pKey1, uint64(nKey1)), util.View(mod, pKey2, uint64(nKey2))))
}

func cbFunc(ctx context.Context, mod api.Module, pCtx, nArg, pArg uint32) {
	module := ctx.Value(moduleKey{}).(*module)
	pApp := uint32(module.call(module.api.userData, uint64(pCtx)))
	args := make([]Value, nArg)
	for i := range args {
		args[i] = Value{
			handle: util.ReadUint32(mod, pArg+ptrlen*uint32(i)),
			module: module,
		}
	}
	context := Context{
		handle: pCtx,
		module: module,
	}
	fn := util.GetHandle(ctx, pApp).(func(ctx Context, arg ...Value))
	fn(context, args...)
}

func cbStep(ctx context.Context, mod api.Module, pCtx, nArg, pArg uint32) {}

func cbFinal(ctx context.Context, mod api.Module, pCtx uint32) {}

func cbValue(ctx context.Context, mod api.Module, pCtx uint32) {}

func cbInverse(ctx context.Context, mod api.Module, pCtx, nArg, pArg uint32) {}
