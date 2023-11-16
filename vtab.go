package sqlite3

import (
	"context"
	"reflect"

	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/tetratelabs/wazero/api"
)

// CreateModule register a new virtual table module name.
func CreateModule[T VTab](conn *Conn, name string, module Module[T]) error {
	var flags int

	const (
		VTAB_CREATOR     = 0x01
		VTAB_DESTROYER   = 0x02
		VTAB_UPDATER     = 0x04
		VTAB_RENAMER     = 0x08
		VTAB_OVERLOADER  = 0x10
		VTAB_CHECKER     = 0x20
		VTAB_TX          = 0x40
		VTAB_SAVEPOINTER = 0x80
	)

	create, ok := reflect.TypeOf(module).MethodByName("Create")
	connect, _ := reflect.TypeOf(module).MethodByName("Connect")
	if ok && create.Type == connect.Type {
		flags |= VTAB_CREATOR
	}

	vtab := connect.Type.Out(0)
	if implements[VTabDestroyer](vtab) {
		flags |= VTAB_DESTROYER
	}
	if implements[VTabUpdater](vtab) {
		flags |= VTAB_UPDATER
	}
	if implements[VTabRenamer](vtab) {
		flags |= VTAB_RENAMER
	}
	if implements[VTabOverloader](vtab) {
		flags |= VTAB_OVERLOADER
	}
	if implements[VTabChecker](vtab) {
		flags |= VTAB_CHECKER
	}
	if implements[VTabTx](vtab) {
		flags |= VTAB_TX
	}
	if implements[VTabSavepointer](vtab) {
		flags |= VTAB_SAVEPOINTER
	}

	defer conn.arena.reset()
	namePtr := conn.arena.string(name)
	modulePtr := util.AddHandle(conn.ctx, module)
	r := conn.call(conn.api.createModule, uint64(conn.handle),
		uint64(namePtr), uint64(flags), uint64(modulePtr))
	return conn.error(r)
}

func implements[T any](typ reflect.Type) bool {
	var ptr *T
	return typ.Implements(reflect.TypeOf(ptr).Elem())
}

func (c *Conn) DeclareVtab(sql string) error {
	defer c.arena.reset()
	sqlPtr := c.arena.string(sql)
	r := c.call(c.api.declareVTab, uint64(c.handle), uint64(sqlPtr))
	return c.error(r)
}

// A Module defines the implementation of a virtual table.
// A Module that doesn't implement [ModuleCreator] provides
// eponymous-only virtual tables or table-valued functions.
//
// https://sqlite.org/c3ref/module.html
type Module[T VTab] interface {
	// https://sqlite.org/vtab.html#xconnect
	Connect(c *Conn, arg ...string) (T, error)
}

// A ModuleCreator allows virtual tables to be created.
// A persistent virtual table must implement [VTabDestroyer].
type ModuleCreator[T VTab] interface {
	Module[T]
	// https://sqlite.org/vtab.html#xcreate
	Create(c *Conn, arg ...string) (T, error)
}

// A VTab describes a particular instance of the virtual table.
//
// https://sqlite.org/c3ref/vtab.html
type VTab interface {
	// https://sqlite.org/vtab.html#xbestindex
	BestIndex(*IndexInfo) error
	// https://sqlite.org/vtab.html#xdisconnect
	Disconnect() error
	// https://sqlite.org/vtab.html#xopen
	Open() (VTabCursor, error)
}

// A VTabDestroyer allows a persistent virtual table to be destroyed.
type VTabDestroyer interface {
	VTab
	// https://sqlite.org/vtab.html#sqlite3_module.xDestroy
	Destroy() error
}

// A VTabUpdater allows a virtual table to be updated.
type VTabUpdater interface {
	VTab
	// https://sqlite.org/vtab.html#xupdate
	Update(arg ...Value) (rowid int64, err error)
}

// A VTabRenamer allows a virtual table to be renamed.
type VTabRenamer interface {
	VTab
	// https://sqlite.org/vtab.html#xrename
	Rename(new string) error
}

// A VTabOverloader allows a virtual table to overload
// SQL functions.
type VTabOverloader interface {
	VTab
	// https://sqlite.org/vtab.html#xfindfunction
	FindFunction(arg int, name string) (func(ctx Context, arg ...Value), IndexConstraintOp)
}

// A VTabChecker allows a virtual table to report errors
// to the PRAGMA integrity_check PRAGMA quick_check commands.
type VTabChecker interface {
	VTab
	// https://sqlite.org/vtab.html#xintegrity
	Integrity(schema, table string, flags int) error
}

// A VTabTx allows a virtual table to implement
// transactions with two-phase commit.
type VTabTx interface {
	VTab
	// https://sqlite.org/vtab.html#xBegin
	Begin() error
	// https://sqlite.org/vtab.html#xsync
	Sync() error
	// https://sqlite.org/vtab.html#xcommit
	Commit() error
	// https://sqlite.org/vtab.html#xrollback
	Rollback() error
}

// A VTabSavepointer allows a virtual table to implement
// nested transactions.
//
// https://sqlite.org/vtab.html#xsavepoint
type VTabSavepointer interface {
	VTabTx
	Savepoint(id int) error
	Release(id int) error
	RollbackTo(id int) error
}

// A VTabCursor describes cursors that point
// into the virtual table and are used
// to loop through the virtual table.
//
// http://sqlite.org/c3ref/vtab_cursor.html
type VTabCursor interface {
	// https://sqlite.org/vtab.html#xclose
	Close() error
	// https://sqlite.org/vtab.html#xfilter
	Filter(idxNum int, idxStr string, arg ...Value) error
	// https://sqlite.org/vtab.html#xnext
	Next() error
	// https://sqlite.org/vtab.html#xeof
	EOF() bool
	// https://sqlite.org/vtab.html#xcolumn
	Column(ctx *Context, n int) error
	// https://sqlite.org/vtab.html#xrowid
	RowID() (int64, error)
}

// An IndexInfo describes virtual table indexing information.
//
// https://sqlite.org/c3ref/index_info.html
type IndexInfo struct {
	/* Inputs */
	Constraint []struct {
		Column int
		Op     IndexConstraintOp
		Usable bool
	}
	OrderBy []struct {
		Column int
		Desc   bool
	}
	/* Outputs */
	ConstraintUsage []struct {
		ArgvIndex int
		Omit      bool
	}
	IdxNum          int
	IdxStr          string
	IdxFlags        IndexScanFlag
	OrderByConsumed bool
	EstimatedCost   float64
	EstimatedRows   int64
	ColumnsUsed     int64
}

// IndexConstraintOp is a virtual table constraint operator code.
//
// https://sqlite.org/c3ref/c_index_constraint_eq.html
type IndexConstraintOp uint8

const (
	Eq        IndexConstraintOp = 2
	Gt        IndexConstraintOp = 4
	Le        IndexConstraintOp = 8
	Lt        IndexConstraintOp = 16
	Ge        IndexConstraintOp = 32
	Match     IndexConstraintOp = 64
	Like      IndexConstraintOp = 65  /* 3.10.0 and later */
	Glob      IndexConstraintOp = 66  /* 3.10.0 and later */
	Regexp    IndexConstraintOp = 67  /* 3.10.0 and later */
	Ne        IndexConstraintOp = 68  /* 3.21.0 and later */
	IsNot     IndexConstraintOp = 69  /* 3.21.0 and later */
	IsNotNull IndexConstraintOp = 70  /* 3.21.0 and later */
	IsNull    IndexConstraintOp = 71  /* 3.21.0 and later */
	Is        IndexConstraintOp = 72  /* 3.21.0 and later */
	Limit     IndexConstraintOp = 73  /* 3.38.0 and later */
	Offset    IndexConstraintOp = 74  /* 3.38.0 and later */
	Function  IndexConstraintOp = 150 /* 3.25.0 and later */
)

// IndexScanFlag is a virtual table scan flag.
//
// https://www.sqlite.org/c3ref/c_index_scan_unique.html
type IndexScanFlag uint8

const (
	Unique IndexScanFlag = 1
)

func vtabConnectCallback(ctx context.Context, mod api.Module, pMod, argc, argv, ppVTab, pzErr uint32) uint32 {
	const handleOffset = 4
	handle := util.ReadUint32(mod, pMod-handleOffset)
	module := util.GetHandle(ctx, handle)
	db := ctx.Value(connKey{}).(*Conn)

	arg := make([]reflect.Value, 1+argc)
	arg[0] = reflect.ValueOf(db)

	for i := uint32(0); i < argc; i++ {
		ptr := util.ReadUint32(mod, argv+i*ptrlen)
		arg[i+1] = reflect.ValueOf(util.ReadString(mod, ptr, _MAX_STRING))
	}

	res := reflect.ValueOf(module).MethodByName("Connect").Call(arg)
	err, _ := res[1].Interface().(error)
	if err == nil {
		handle := util.AddHandle(ctx, res[0].Interface())
		ptr := util.ReadUint32(mod, ppVTab)
		util.WriteUint32(mod, ptr-handleOffset, handle)
		return _OK
	}

	// TODO: error message
	return errorCode(err, ERROR)
}

func vtabBestIndexCallback(ctx context.Context, mod api.Module, pVTab, pIdxInfo uint32) uint32 {
	const handleOffset = 4
	handle := util.ReadUint32(mod, pVTab-handleOffset)
	vtab := util.GetHandle(ctx, handle).(VTab)
	_ = vtab
	return 1
}

func vtabDisconnectCallback(ctx context.Context, mod api.Module, pVTab uint32) uint32 {
	return 1
}

func vtabIntegrityCallback(ctx context.Context, mod api.Module, pVTab, zSchema, zTabName, mFlags, pzErr uint32) uint32 {
	return 1
}

func vtabCallbackI(ctx context.Context, mod api.Module, _ uint32) uint32 {
	return 1
}

func vtabCallbackII(ctx context.Context, mod api.Module, _, _ uint32) uint32 {
	return 1
}

func vtabCallbackIIII(ctx context.Context, mod api.Module, _, _, _, _ uint32) uint32 {
	return 1
}

func cursorOpenCallback(ctx context.Context, mod api.Module, pVTab, ppCur uint32) uint32 {
	return 1
}

func cursorFilterCallback(ctx context.Context, mod api.Module, pCur, idxNum, idxStr, argc, argv uint32) uint32 {
	return 1
}

func cursorColumnCallback(ctx context.Context, mod api.Module, pCur, pCtx, n uint32) uint32 {
	return 1
}

func cursorRowidCallback(ctx context.Context, mod api.Module, pCur, pRowid uint32) uint32 {
	return 1
}

func cursorCallbackI(ctx context.Context, mod api.Module, _ uint32) uint32 {
	return 1
}
