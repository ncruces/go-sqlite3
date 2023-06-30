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

func cbDestroy(ctx context.Context, mod api.Module, pArg uint32) {
	util.DelHandle(ctx, pArg)
}

func cbCompare(ctx context.Context, mod api.Module, pArg, nKey1, pKey1, nKey2, pKey2 uint32) uint32 {
	fn := util.GetHandle(ctx, pArg).(func(a, b []byte) int)
	return uint32(fn(util.View(mod, pKey1, uint64(nKey1)), util.View(mod, pKey2, uint64(nKey2))))
}

func cbFunc(ctx context.Context, mod api.Module, pCtx, nArg, pArg uint32) {}

func cbStep(ctx context.Context, mod api.Module, pCtx, nArg, pArg uint32) {}

func cbFinal(ctx context.Context, mod api.Module, pCtx uint32) {}

func cbValue(ctx context.Context, mod api.Module, pCtx uint32) {}

func cbInverse(ctx context.Context, mod api.Module, pCtx, nArg, pArg uint32) {}
