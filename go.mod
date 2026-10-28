module github.com/ncruces/go-sqlite3

go 1.26.0

require (
	github.com/ncruces/go-sqlite3-wasm/v6 v6.3.35401
	github.com/ncruces/julianday v1.0.0
	github.com/ncruces/sort v1.0.3
	github.com/ncruces/wbt v1.0.0
	golang.org/x/sys v0.49.0
)

require (
	github.com/dchest/siphash v1.2.3 // ext/bloom
	github.com/google/uuid v1.6.0 // ext/uuid
	github.com/psanford/httpreadat v0.1.0 // example
	github.com/segmentio/go-hll v1.0.1 // ext/approx
	github.com/twmb/murmur3 v1.1.8 // ext/approx
	golang.org/x/crypto v0.58.0 // vfs/adiantum vfs/xts
	golang.org/x/sync v0.24.0 // test
	golang.org/x/text v0.43.0 // ext/unicode
	lukechampine.com/adiantum v1.1.1 // vfs/adiantum
)

require github.com/pkg/errors v0.8.0 // indirect

retract (
	v0.23.2 // tagged from the wrong branch
	v0.4.0 // tagged from the wrong branch
)
