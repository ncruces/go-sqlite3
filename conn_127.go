//go:build go1.27

package sqlite3

// CreateModule registers a new virtual table module name.
// If create is nil, the virtual table is eponymous.
//
// https://sqlite.org/c3ref/create_module.html
func (c *Conn) CreateModule[T VTab](name string, create, connect VTabConstructor[T]) error {
	return CreateModule(c, name, create, connect)
}

// ExtensionInit loads an SQLite extension library.
//
// https://sqlite.org/loadext.html
func (c *Conn) ExtensionInit[Env any, Mod ExtensionLibrary](init func(env Env) Mod, info ExtensionInfo) error {
	return ExtensionInit(c, init, info)
}
