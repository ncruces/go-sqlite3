package sqlite3

type Module interface {
	Connect(db *Conn, arg ...string) (Vtab, error)
}

type ModuleCreator interface {
	Module
	Create(db *Conn, arg ...string) (Vtab, error)
}

type ModuleShadowNamer interface {
	Module
	ShadowName(suffix string) bool
}

type Vtab interface {
	BestIndex(*IndexInfo) error
	Disconnect() error
	Destroy() error
	Open() (VtabCursor, error)
}

type VtabUpdater interface {
	Vtab
	Update(arg ...Value) (rowid int64, err error)
}

type VtabRenamer interface {
	Vtab
	Rename(new string) error
}

type VtabOverloader interface {
	Vtab
	FindFunction(arg int, name string) (func(ctx Context, arg ...Value), error)
}

type VtabChecker interface {
	Vtab
	Integrity(schema, table string, flags int) error
}

type VtabTx interface {
	Vtab
	Begin() error
	Sync() error
	Commit() error
	Rollback() error
}

type VtabSavepointer interface {
	VtabTx
	Savepoint(n int) error
	Release(n int) error
	RollbackTo(n int) error
}

type VtabCursor interface {
	Close() error
	Filter(idxNum int, idxStr string, arg ...Value)
	Next() error
	Eof() bool
	Column(ctx *Context, n int) error
	Rowid() (int64, error)
}

type IndexInfo struct{}
