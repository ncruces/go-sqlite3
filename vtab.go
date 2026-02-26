package sqlite3

import (
	"errors"
	"reflect"

	"github.com/ncruces/go-sqlite3/internal/util"
)

// CreateModule registers a new virtual table module name.
// If create is nil, the virtual table is eponymous.
//
// https://sqlite.org/c3ref/create_module.html
func CreateModule[T VTab](db *Conn, name string, create, connect VTabConstructor[T]) error {
	var flags int

	const (
		VTAB_CREATOR     = 0x001
		VTAB_DESTROYER   = 0x002
		VTAB_UPDATER     = 0x004
		VTAB_RENAMER     = 0x008
		VTAB_OVERLOADER  = 0x010
		VTAB_CHECKER     = 0x020
		VTAB_TXN         = 0x040
		VTAB_SAVEPOINTER = 0x080
		VTAB_SHADOWTABS  = 0x100
	)

	if create != nil {
		flags |= VTAB_CREATOR
	}

	vtab := reflect.TypeOf(connect).Out(0)
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
	if implements[VTabTxn](vtab) {
		flags |= VTAB_TXN
	}
	if implements[VTabSavepointer](vtab) {
		flags |= VTAB_SAVEPOINTER
	}
	if implements[VTabShadowTabler](vtab) {
		flags |= VTAB_SHADOWTABS
	}

	var modulePtr ptr_t
	defer db.arena.mark()()
	namePtr := db.arena.string(name)
	if connect != nil {
		modulePtr = util.AddHandle(db.ctx, module[T]{create, connect})
	}
	rc := res_t(db.mod.Xsqlite3_create_module_go(int32(db.handle),
		int32(namePtr), int32(flags), int32(modulePtr)))
	return db.error(rc)
}

func implements[T any](typ reflect.Type) bool {
	var ptr *T
	return typ.Implements(reflect.TypeOf(ptr).Elem())
}

// DeclareVTab declares the schema of a virtual table.
//
// https://sqlite.org/c3ref/declare_vtab.html
func (c *Conn) DeclareVTab(sql string) error {
	if c.interrupt.Err() != nil {
		return INTERRUPT
	}
	defer c.arena.mark()()
	textPtr := c.arena.string(sql)
	rc := res_t(c.mod.Xsqlite3_declare_vtab(int32(c.handle), int32(textPtr)))
	return c.error(rc)
}

// VTabConflictMode is a virtual table conflict resolution mode.
//
// https://sqlite.org/c3ref/c_fail.html
type VTabConflictMode uint8

const (
	VTAB_ROLLBACK VTabConflictMode = 1
	VTAB_IGNORE   VTabConflictMode = 2
	VTAB_FAIL     VTabConflictMode = 3
	VTAB_ABORT    VTabConflictMode = 4
	VTAB_REPLACE  VTabConflictMode = 5
)

// VTabOnConflict determines the virtual table conflict policy.
//
// https://sqlite.org/c3ref/vtab_on_conflict.html
func (c *Conn) VTabOnConflict() VTabConflictMode {
	return VTabConflictMode(c.mod.Xsqlite3_vtab_on_conflict(int32(c.handle)))
}

// VTabConfigOption is a virtual table configuration option.
//
// https://sqlite.org/c3ref/c_vtab_constraint_support.html
type VTabConfigOption uint8

const (
	VTAB_CONSTRAINT_SUPPORT VTabConfigOption = 1
	VTAB_INNOCUOUS          VTabConfigOption = 2
	VTAB_DIRECTONLY         VTabConfigOption = 3
	VTAB_USES_ALL_SCHEMAS   VTabConfigOption = 4
)

// VTabConfig configures various facets of the virtual table interface.
//
// https://sqlite.org/c3ref/vtab_config.html
func (c *Conn) VTabConfig(op VTabConfigOption, args ...any) error {
	var i int32
	if op == VTAB_CONSTRAINT_SUPPORT && len(args) > 0 {
		if b, ok := args[0].(bool); ok && b {
			i = 1
		}
	}
	rc := res_t(c.mod.Xsqlite3_vtab_config_go(int32(c.handle), int32(op), i))
	return c.error(rc)
}

// VTabConstructor is a virtual table constructor function.
type VTabConstructor[T VTab] func(db *Conn, module, schema, table string, arg ...string) (T, error)

type module[T VTab] [2]VTabConstructor[T]

type vtabConstructor int

const (
	xCreate  vtabConstructor = 0
	xConnect vtabConstructor = 1
)

// A VTab describes a particular instance of the virtual table.
// A VTab may optionally implement [io.Closer] to free resources.
//
// https://sqlite.org/c3ref/vtab.html
type VTab interface {
	// https://sqlite.org/vtab.html#xbestindex
	BestIndex(*IndexInfo) error
	// https://sqlite.org/vtab.html#xopen
	Open() (VTabCursor, error)
}

// A VTabDestroyer allows a virtual table to drop persistent state.
type VTabDestroyer interface {
	VTab
	// https://sqlite.org/vtab.html#sqlite3_module.xDestroy
	Destroy() error
}

// A VTabUpdater allows a virtual table to be updated.
// Implementations must not retain arg.
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

// A VTabOverloader allows a virtual table to overload SQL functions.
type VTabOverloader interface {
	VTab
	// https://sqlite.org/vtab.html#xfindfunction
	FindFunction(arg int, name string) (ScalarFunction, IndexConstraintOp)
}

// A VTabShadowTabler allows a virtual table to protect the content
// of shadow tables from being corrupted by hostile SQL.
//
// Implementing this interface signals that a virtual table named
// "mumble" reserves all table names starting with "mumble_".
type VTabShadowTabler interface {
	VTab
	// https://sqlite.org/vtab.html#the_xshadowname_method
	ShadowTables()
}

// A VTabChecker allows a virtual table to report errors
// to the PRAGMA integrity_check and PRAGMA quick_check commands.
//
// Integrity should return an error if it finds problems in the content of the virtual table,
// but should avoid returning a (wrapped) [Error], [ErrorCode] or [ExtendedErrorCode],
// as those indicate the Integrity method itself encountered problems
// while trying to evaluate the virtual table content.
type VTabChecker interface {
	VTab
	// https://sqlite.org/vtab.html#xintegrity
	Integrity(schema, table string, flags int) error
}

// A VTabTxn allows a virtual table to implement
// transactions with two-phase commit.
//
// Anything that is required as part of a commit that may fail
// should be performed in the Sync() callback.
// Current versions of SQLite ignore any errors
// returned by Commit() and Rollback().
type VTabTxn interface {
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
	VTabTxn
	Savepoint(id int) error
	Release(id int) error
	RollbackTo(id int) error
}

// A VTabCursor describes cursors that point
// into the virtual table and are used
// to loop through the virtual table.
// A VTabCursor may optionally implement
// [io.Closer] to free resources.
// Implementations of Filter must not retain arg.
//
// https://sqlite.org/c3ref/vtab_cursor.html
type VTabCursor interface {
	// https://sqlite.org/vtab.html#xfilter
	Filter(idxNum int, idxStr string, arg ...Value) error
	// https://sqlite.org/vtab.html#xnext
	Next() error
	// https://sqlite.org/vtab.html#xeof
	EOF() bool
	// https://sqlite.org/vtab.html#xcolumn
	Column(ctx Context, n int) error
	// https://sqlite.org/vtab.html#xrowid
	RowID() (int64, error)
}

// An IndexInfo describes virtual table indexing information.
//
// https://sqlite.org/c3ref/index_info.html
type IndexInfo struct {
	// Inputs
	Constraint  []IndexConstraint
	OrderBy     []IndexOrderBy
	ColumnsUsed uint64
	// Outputs
	ConstraintUsage []IndexConstraintUsage
	IdxNum          int
	IdxStr          string
	IdxFlags        IndexScanFlag
	OrderByConsumed bool
	EstimatedCost   float64
	EstimatedRows   int64
	// Internal
	c      *Conn
	handle ptr_t
}

// An IndexConstraint describes virtual table indexing constraint information.
//
// https://sqlite.org/c3ref/index_info.html
type IndexConstraint struct {
	Column int
	Op     IndexConstraintOp
	Usable bool
}

// An IndexOrderBy describes virtual table indexing order by information.
//
// https://sqlite.org/c3ref/index_info.html
type IndexOrderBy struct {
	Column int
	Desc   bool
}

// An IndexConstraintUsage describes how virtual table indexing constraints will be used.
//
// https://sqlite.org/c3ref/index_info.html
type IndexConstraintUsage struct {
	ArgvIndex int
	Omit      bool
}

// RHSValue returns the value of the right-hand operand of a constraint
// if the right-hand operand is known.
//
// https://sqlite.org/c3ref/vtab_rhs_value.html
func (idx *IndexInfo) RHSValue(column int) (Value, error) {
	defer idx.c.arena.mark()()
	valPtr := idx.c.arena.new(ptrlen)
	rc := res_t(idx.c.mod.Xsqlite3_vtab_rhs_value(int32(idx.handle),
		int32(column), int32(valPtr)))
	if err := idx.c.error(rc); err != nil {
		return Value{}, err
	}
	return Value{
		c:      idx.c,
		handle: util.Read32[ptr_t](idx.c.mod, valPtr),
	}, nil
}

// Collation returns the name of the collation for a virtual table constraint.
//
// https://sqlite.org/c3ref/vtab_collation.html
func (idx *IndexInfo) Collation(column int) string {
	ptr := ptr_t(idx.c.mod.Xsqlite3_vtab_collation(int32(idx.handle),
		int32(column)))
	return util.ReadString(idx.c.mod, ptr, _MAX_NAME)
}

// Distinct determines if a virtual table query is DISTINCT.
//
// https://sqlite.org/c3ref/vtab_distinct.html
func (idx *IndexInfo) Distinct() int {
	i := int32(idx.c.mod.Xsqlite3_vtab_distinct(int32(idx.handle)))
	return int(i)
}

// In identifies and handles IN constraints.
//
// https://sqlite.org/c3ref/vtab_in.html
func (idx *IndexInfo) In(column, handle int) bool {
	b := int32(idx.c.mod.Xsqlite3_vtab_in(int32(idx.handle),
		int32(column), int32(handle)))
	return b != 0
}

func (idx *IndexInfo) load() {
	// https://sqlite.org/c3ref/index_info.html
	mod := idx.c.mod
	ptr := idx.handle

	nConstraint := util.Read32[int32](mod, ptr+0)
	idx.Constraint = make([]IndexConstraint, nConstraint)
	idx.ConstraintUsage = make([]IndexConstraintUsage, nConstraint)
	idx.OrderBy = make([]IndexOrderBy, util.Read32[int32](mod, ptr+8))

	constraintPtr := util.Read32[ptr_t](mod, ptr+4)
	constraint := idx.Constraint
	for i := range idx.Constraint {
		constraint[i] = IndexConstraint{
			Column: int(util.Read32[int32](mod, constraintPtr+0)),
			Op:     util.Read[IndexConstraintOp](mod, constraintPtr+4),
			Usable: util.Read[byte](mod, constraintPtr+5) != 0,
		}
		constraintPtr += 12
	}

	orderByPtr := util.Read32[ptr_t](mod, ptr+12)
	orderBy := idx.OrderBy
	for i := range orderBy {
		orderBy[i] = IndexOrderBy{
			Column: int(util.Read32[int32](mod, orderByPtr+0)),
			Desc:   util.Read[byte](mod, orderByPtr+4) != 0,
		}
		orderByPtr += 8
	}

	idx.EstimatedCost = util.ReadFloat64(mod, ptr+40)
	idx.EstimatedRows = util.Read64[int64](mod, ptr+48)
	idx.ColumnsUsed = util.Read64[uint64](mod, ptr+64)
}

func (idx *IndexInfo) save() {
	// https://sqlite.org/c3ref/index_info.html
	mod := idx.c.mod
	ptr := idx.handle

	usagePtr := util.Read32[ptr_t](mod, ptr+16)
	for _, usage := range idx.ConstraintUsage {
		util.Write32(mod, usagePtr+0, int32(usage.ArgvIndex))
		if usage.Omit {
			util.Write(mod, usagePtr+4, int8(1))
		}
		usagePtr += 8
	}

	util.Write32(mod, ptr+20, int32(idx.IdxNum))
	if idx.IdxStr != "" {
		util.Write32(mod, ptr+24, idx.c.newString(idx.IdxStr))
		util.WriteBool(mod, ptr+28, true) // needToFreeIdxStr
	}
	if idx.OrderByConsumed {
		util.WriteBool(mod, ptr+32, true)
	}
	util.WriteFloat64(mod, ptr+40, idx.EstimatedCost)
	util.Write64(mod, ptr+48, idx.EstimatedRows)
	util.Write32(mod, ptr+56, idx.IdxFlags)
}

// IndexConstraintOp is a virtual table constraint operator code.
//
// https://sqlite.org/c3ref/c_index_constraint_eq.html
type IndexConstraintOp uint8

const (
	INDEX_CONSTRAINT_EQ        IndexConstraintOp = 2
	INDEX_CONSTRAINT_GT        IndexConstraintOp = 4
	INDEX_CONSTRAINT_LE        IndexConstraintOp = 8
	INDEX_CONSTRAINT_LT        IndexConstraintOp = 16
	INDEX_CONSTRAINT_GE        IndexConstraintOp = 32
	INDEX_CONSTRAINT_MATCH     IndexConstraintOp = 64
	INDEX_CONSTRAINT_LIKE      IndexConstraintOp = 65
	INDEX_CONSTRAINT_GLOB      IndexConstraintOp = 66
	INDEX_CONSTRAINT_REGEXP    IndexConstraintOp = 67
	INDEX_CONSTRAINT_NE        IndexConstraintOp = 68
	INDEX_CONSTRAINT_ISNOT     IndexConstraintOp = 69
	INDEX_CONSTRAINT_ISNOTNULL IndexConstraintOp = 70
	INDEX_CONSTRAINT_ISNULL    IndexConstraintOp = 71
	INDEX_CONSTRAINT_IS        IndexConstraintOp = 72
	INDEX_CONSTRAINT_LIMIT     IndexConstraintOp = 73
	INDEX_CONSTRAINT_OFFSET    IndexConstraintOp = 74
	INDEX_CONSTRAINT_FUNCTION  IndexConstraintOp = 150
)

// IndexScanFlag is a virtual table scan flag.
//
// https://sqlite.org/c3ref/c_index_scan_unique.html
type IndexScanFlag uint32

const (
	INDEX_SCAN_UNIQUE IndexScanFlag = 0x00000001
	INDEX_SCAN_HEX    IndexScanFlag = 0x00000002
)

func (sqlt *sqlite) vtabModuleCallback(kind vtabConstructor, pMod, nArg, pArg, ppVTab, pzErr int32) int32 {
	arg := make([]reflect.Value, 1+nArg)
	arg[0] = reflect.ValueOf(sqlt.ctx.Value(connKey{}))

	for i := range nArg {
		ptr := util.Read32[ptr_t](sqlt.mod, ptr_t(pArg+i*ptrlen))
		arg[i+1] = reflect.ValueOf(util.ReadString(sqlt.mod, ptr, _MAX_SQL_LENGTH))
	}

	module := sqlt.vtabGetHandle(pMod)
	val := reflect.ValueOf(module).Index(int(kind)).Call(arg)
	err, _ := val[1].Interface().(error)
	if err == nil {
		sqlt.vtabPutHandle(ppVTab, val[0].Interface())
	}

	return sqlt.vtabError(pzErr, _PTR_ERROR, err, ERROR)
}

func (sqlt *sqlite) Xgo_vtab_create(pMod, nArg, pArg, ppVTab, pzErr int32) int32 {
	return sqlt.vtabModuleCallback(xCreate, pMod, nArg, pArg, ppVTab, pzErr)
}

func (sqlt *sqlite) Xgo_vtab_connect(pMod, nArg, pArg, ppVTab, pzErr int32) int32 {
	return sqlt.vtabModuleCallback(xConnect, pMod, nArg, pArg, ppVTab, pzErr)
}

func (sqlt *sqlite) Xgo_vtab_disconnect(pVTab int32) int32 {
	err := sqlt.vtabDelHandle(pVTab)
	return sqlt.vtabError(0, _PTR_ERROR, err, ERROR)
}

func (sqlt *sqlite) Xgo_vtab_destroy(pVTab int32) int32 {
	vtab := sqlt.vtabGetHandle(pVTab).(VTabDestroyer)
	err := errors.Join(vtab.Destroy(), sqlt.vtabDelHandle(pVTab))
	return sqlt.vtabError(0, _PTR_ERROR, err, ERROR)
}

func (sqlt *sqlite) Xgo_vtab_best_index(pVTab, pIdxInfo int32) int32 {
	var info IndexInfo
	info.handle = ptr_t(pIdxInfo)
	info.c = sqlt.ctx.Value(connKey{}).(*Conn)
	info.load()

	vtab := sqlt.vtabGetHandle(pVTab).(VTab)
	err := vtab.BestIndex(&info)

	info.save()
	return sqlt.vtabError(pVTab, _VTAB_ERROR, err, ERROR)
}

func (sqlt *sqlite) Xgo_vtab_update(pVTab, nArg, pArg, pRowID int32) int32 {
	db := sqlt.ctx.Value(connKey{}).(*Conn)
	args := callbackArgs(db, nArg, ptr_t(pArg))
	defer returnArgs(args)

	vtab := sqlt.vtabGetHandle(pVTab).(VTabUpdater)
	rowID, err := vtab.Update(*args...)
	if err == nil {
		util.Write64(sqlt.mod, ptr_t(pRowID), rowID)
	}

	return sqlt.vtabError(pVTab, _VTAB_ERROR, err, ERROR)
}

func (sqlt *sqlite) Xgo_vtab_rename(pVTab, zNew int32) int32 {
	vtab := sqlt.vtabGetHandle(pVTab).(VTabRenamer)
	err := vtab.Rename(util.ReadString(sqlt.mod, ptr_t(zNew), _MAX_NAME))
	return sqlt.vtabError(pVTab, _VTAB_ERROR, err, ERROR)
}

func (sqlt *sqlite) Xgo_vtab_find_function(pVTab, nArg, zName, pxFunc int32) int32 {
	vtab := sqlt.vtabGetHandle(pVTab).(VTabOverloader)
	f, op := vtab.FindFunction(int(nArg), util.ReadString(sqlt.mod, ptr_t(zName), _MAX_NAME))
	if op != 0 {
		var wrapper ptr_t
		wrapper = util.AddHandle(sqlt.ctx, func(c Context, arg ...Value) {
			defer util.DelHandle(sqlt.ctx, wrapper)
			f(c, arg...)
		})
		util.Write32(sqlt.mod, ptr_t(pxFunc), wrapper)
	}
	return int32(op)
}

func (sqlt *sqlite) Xgo_vtab_integrity(pVTab, zSchema, zTabName, mFlags, pzErr int32) int32 {
	vtab := sqlt.vtabGetHandle(pVTab).(VTabChecker)
	schema := util.ReadString(sqlt.mod, ptr_t(zSchema), _MAX_NAME)
	table := util.ReadString(sqlt.mod, ptr_t(zTabName), _MAX_NAME)
	err := vtab.Integrity(schema, table, int(uint32(mFlags)))
	// xIntegrity should return OK - even if it finds problems in the content of the virtual table.
	// https://sqlite.org/vtab.html#xintegrity
	return sqlt.vtabError(pzErr, _PTR_ERROR, err, _OK)
}

func (sqlt *sqlite) Xgo_vtab_begin(pVTab int32) int32 {
	vtab := sqlt.vtabGetHandle(pVTab).(VTabTxn)
	err := vtab.Begin()
	return sqlt.vtabError(pVTab, _VTAB_ERROR, err, ERROR)
}

func (sqlt *sqlite) Xgo_vtab_sync(pVTab int32) int32 {
	vtab := sqlt.vtabGetHandle(pVTab).(VTabTxn)
	err := vtab.Sync()
	return sqlt.vtabError(pVTab, _VTAB_ERROR, err, ERROR)
}

func (sqlt *sqlite) Xgo_vtab_commit(pVTab int32) int32 {
	vtab := sqlt.vtabGetHandle(pVTab).(VTabTxn)
	err := vtab.Commit()
	return sqlt.vtabError(pVTab, _VTAB_ERROR, err, ERROR)
}

func (sqlt *sqlite) Xgo_vtab_rollback(pVTab int32) int32 {
	vtab := sqlt.vtabGetHandle(pVTab).(VTabTxn)
	err := vtab.Rollback()
	return sqlt.vtabError(pVTab, _VTAB_ERROR, err, ERROR)
}

func (sqlt *sqlite) Xgo_vtab_savepoint(pVTab, id int32) int32 {
	vtab := sqlt.vtabGetHandle(pVTab).(VTabSavepointer)
	err := vtab.Savepoint(int(id))
	return sqlt.vtabError(pVTab, _VTAB_ERROR, err, ERROR)
}

func (sqlt *sqlite) Xgo_vtab_release(pVTab, id int32) int32 {
	vtab := sqlt.vtabGetHandle(pVTab).(VTabSavepointer)
	err := vtab.Release(int(id))
	return sqlt.vtabError(pVTab, _VTAB_ERROR, err, ERROR)
}

func (sqlt *sqlite) Xgo_vtab_rollback_to(pVTab, id int32) int32 {
	vtab := sqlt.vtabGetHandle(pVTab).(VTabSavepointer)
	err := vtab.RollbackTo(int(id))
	return sqlt.vtabError(pVTab, _VTAB_ERROR, err, ERROR)
}

func (sqlt *sqlite) Xgo_cur_open(pVTab, ppCur int32) int32 {
	vtab := sqlt.vtabGetHandle(pVTab).(VTab)

	cursor, err := vtab.Open()
	if err == nil {
		sqlt.vtabPutHandle(ppCur, cursor)
	}

	return sqlt.vtabError(pVTab, _VTAB_ERROR, err, ERROR)
}

func (sqlt *sqlite) Xgo_cur_close(pCur int32) int32 {
	err := sqlt.vtabDelHandle(pCur)
	return sqlt.vtabError(0, _PTR_ERROR, err, ERROR)
}

func (sqlt *sqlite) Xgo_cur_filter(pCur, idxNum, idxStr, nArg, pArg int32) int32 {
	db := sqlt.ctx.Value(connKey{}).(*Conn)
	args := callbackArgs(db, nArg, ptr_t(pArg))
	defer returnArgs(args)

	var idxName string
	if idxStr != 0 {
		idxName = util.ReadString(sqlt.mod, ptr_t(idxStr), _MAX_LENGTH)
	}

	cursor := sqlt.vtabGetHandle(pCur).(VTabCursor)
	err := cursor.Filter(int(idxNum), idxName, *args...)
	return sqlt.vtabError(pCur, _CURSOR_ERROR, err, ERROR)
}

func (sqlt *sqlite) Xgo_cur_eof(pCur int32) int32 {
	cursor := sqlt.vtabGetHandle(pCur).(VTabCursor)
	if cursor.EOF() {
		return 1
	}
	return 0
}

func (sqlt *sqlite) Xgo_cur_next(pCur int32) int32 {
	cursor := sqlt.vtabGetHandle(pCur).(VTabCursor)
	err := cursor.Next()
	return sqlt.vtabError(pCur, _CURSOR_ERROR, err, ERROR)
}

func (sqlt *sqlite) Xgo_cur_column(pCur, pCtx, n int32) int32 {
	cursor := sqlt.vtabGetHandle(pCur).(VTabCursor)
	db := sqlt.ctx.Value(connKey{}).(*Conn)
	err := cursor.Column(Context{db, ptr_t(pCtx)}, int(n))
	return sqlt.vtabError(pCur, _CURSOR_ERROR, err, ERROR)
}

func (sqlt *sqlite) Xgo_cur_rowid(pCur, pRowID int32) int32 {
	cursor := sqlt.vtabGetHandle(pCur).(VTabCursor)

	rowID, err := cursor.RowID()
	if err == nil {
		util.Write64(sqlt.mod, ptr_t(pRowID), rowID)
	}

	return sqlt.vtabError(pCur, _CURSOR_ERROR, err, ERROR)
}

const (
	_PTR_ERROR = iota
	_VTAB_ERROR
	_CURSOR_ERROR
)

func (sqlt *sqlite) vtabError(ptr int32, kind uint32, err error, def ErrorCode) int32 {
	const zErrMsgOffset = 8
	msg, code := errorCode(err, def)
	if ptr != 0 && msg != "" {
		switch kind {
		case _VTAB_ERROR:
			ptr = ptr + zErrMsgOffset // zErrMsg
		case _CURSOR_ERROR:
			ptr = int32(util.Read32[ptr_t](sqlt.mod, ptr_t(ptr))) + zErrMsgOffset // pVTab->zErrMsg
		}
		db := sqlt.ctx.Value(connKey{}).(*Conn)
		if ptr := util.Read32[ptr_t](sqlt.mod, ptr_t(ptr)); ptr != 0 {
			db.free(ptr)
		}
		util.Write32(sqlt.mod, ptr_t(ptr), db.newString(msg))
	}
	return int32(code)
}

func (sqlt *sqlite) vtabGetHandle(ptr int32) any {
	const handleOffset = 4
	handle := util.Read32[ptr_t](sqlt.mod, ptr_t(ptr)-handleOffset)
	return util.GetHandle(sqlt.ctx, handle)
}

func (sqlt *sqlite) vtabDelHandle(ptr int32) error {
	const handleOffset = 4
	handle := util.Read32[ptr_t](sqlt.mod, ptr_t(ptr)-handleOffset)
	return util.DelHandle(sqlt.ctx, handle)
}

func (sqlt *sqlite) vtabPutHandle(pptr int32, val any) {
	const handleOffset = 4
	handle := util.AddHandle(sqlt.ctx, val)
	ptr := util.Read32[ptr_t](sqlt.mod, ptr_t(pptr))
	util.Write32(sqlt.mod, ptr-handleOffset, handle)
}
