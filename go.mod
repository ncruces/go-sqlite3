module github.com/ncruces/go-sqlite3

go 1.21

require (
	github.com/ncruces/julianday v1.0.0
	github.com/psanford/httpreadat v0.1.0
	github.com/tetratelabs/wazero v1.7.0
	golang.org/x/crypto v0.22.0
	golang.org/x/sync v0.7.0
	golang.org/x/sys v0.19.0
	golang.org/x/text v0.14.0
)

retract v0.4.0 // tagged from the wrong branch

replace github.com/tetratelabs/wazero => github.com/ncruces/wazero v0.0.0-20240409112346-f2be6da37a3d
