package sqlite3

// A Module defines the implementation of a virtual table.
// Modules that don't also implement [ModuleCreator] provide
// eponymous-only virtual tables or table-valued functions.
//
// https://sqlite.org/c3ref/module.html
type Module interface {
	// https://sqlite.org/vtab.html#xconnect
	Connect(db *Conn, arg ...string) (VTab, error)
}

// A ModuleCreator extends Module for
// non-eponymous virtual tables.
type ModuleCreator interface {
	Module
	// https://sqlite.org/vtab.html#xcreate
	Create(db *Conn, arg ...string) (VTabDestroyer, error)
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

// A VTabDestroyer allows a virtual table to be destroyed.
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
