package sqlite3

import "sync"

var (
	// +checklocks:extRegistryMtx
	extRegistry    []func(*Conn) error
	extRegistryMtx sync.RWMutex
)

// AutoExtension causes the entryPoint function to be invoked
// for each new database connection that is created.
//
// https://sqlite.org/c3ref/auto_extension.html
func AutoExtension(entryPoint func(*Conn) error) {
	extRegistryMtx.Lock()
	defer extRegistryMtx.Unlock()
	extRegistry = append(extRegistry, entryPoint)
}

func allExtensions(yield func(func(*Conn) error) bool) {
	extRegistryMtx.RLock()
	defer extRegistryMtx.RUnlock()
	for _, ext := range extRegistry {
		if !yield(ext) {
			return
		}
	}
}
