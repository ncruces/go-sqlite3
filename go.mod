module github.com/ncruces/go-sqlite3

go 1.24.0

require (
	github.com/ncruces/julianday v1.0.0
	github.com/ncruces/sort v0.1.6
	github.com/ncruces/wbt v1.0.0
	github.com/tetratelabs/wazero v1.11.0
	golang.org/x/sys v0.40.0
)

require (
	github.com/dchest/siphash v1.2.3 // ext/bloom
	github.com/google/uuid v1.6.0 // ext/uuid
	github.com/psanford/httpreadat v0.1.0 // example
	golang.org/x/crypto v0.46.0 // vfs/adiantum vfs/xts
	golang.org/x/sync v0.19.0 // test
	golang.org/x/text v0.32.0 // ext/unicode
	lukechampine.com/adiantum v1.1.1 // vfs/adiantum
)

retract (
	v0.23.2 // tagged from the wrong branch
	v0.4.0 // tagged from the wrong branch
)
