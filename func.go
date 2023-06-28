package sqlite3

import (
	"context"

	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

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

func cbDestroy(ctx context.Context, mod api.Module, pArg uint32) {}

func cbCompare(ctx context.Context, mod api.Module, pArg, nKey1, pKey1, nKey2, pKey2 uint32) uint32 {
	return 0
}

func cbFunc(ctx context.Context, mod api.Module, pCtx, nArg, pArg uint32) {}

func cbStep(ctx context.Context, mod api.Module, pCtx, nArg, pArg uint32) {}

func cbFinal(ctx context.Context, mod api.Module, pCtx uint32) {}

func cbValue(ctx context.Context, mod api.Module, pCtx uint32) {}

func cbInverse(ctx context.Context, mod api.Module, pCtx, nArg, pArg uint32) {}
