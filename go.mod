module github.com/ncruces/go-sqlite3

go 1.21

toolchain go1.23.0

require (
	github.com/dchest/siphash v1.2.3
	github.com/google/uuid v1.6.0
	github.com/ncruces/julianday v1.0.0
	github.com/ncruces/sort v0.1.2
	github.com/psanford/httpreadat v0.1.0
	github.com/tetratelabs/wazero v1.8.0
	golang.org/x/crypto v0.27.0
	golang.org/x/sync v0.8.0
	golang.org/x/sys v0.26.0
	golang.org/x/text v0.18.0
	lukechampine.com/adiantum v1.1.1
)

retract v0.4.0 // tagged from the wrong branch
