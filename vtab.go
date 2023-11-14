package sqlite3

// https://sqlite.org/vtab.html#xconnect
type Module interface {
	Connect(db *Conn, arg ...string) (VTab, error)
}

// https://sqlite.org/vtab.html#xcreate
type ModuleCreator interface {
	Module
	Create(db *Conn, arg ...string) (VTabDestroyer, error)
}

type VTab interface {
	// https://sqlite.org/vtab.html#xbestindex
	BestIndex(*IndexInfo) error
	// https://sqlite.org/vtab.html#xdisconnect
	Disconnect() error
	// https://sqlite.org/vtab.html#xopen
	Open() (VTabCursor, error)
}

// https://sqlite.org/vtab.html#sqlite3_module.xDestroy
type VTabDestroyer interface {
	VTab
	Destroy() error
}

// https://sqlite.org/vtab.html#xupdate
type VTabUpdater interface {
	VTab
	Update(arg ...Value) (rowid int64, err error)
}

// https://sqlite.org/vtab.html#xrename
type VTabRenamer interface {
	VTab
	Rename(new string) error
}

// https://sqlite.org/vtab.html#xfindfunction
type VTabOverloader interface {
	VTab
	FindFunction(arg int, name string) (func(ctx Context, arg ...Value), IndexConstraint)
}

// https://sqlite.org/vtab.html#xintegrity
type VTabChecker interface {
	VTab
	Integrity(schema, table string, flags int) error
}

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

// https://sqlite.org/vtab.html#xsavepoint
type VTabSavepointer interface {
	VTabTx
	Savepoint(id int) error
	Release(id int) error
	RollbackTo(id int) error
}

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

type IndexInfo struct{}

type IndexConstraint uint8
