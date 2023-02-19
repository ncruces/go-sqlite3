package sqlite3

// Return true if stmt is an empty SQL statement.
// This is used as an optimization.
// It's OK to always return false here.
func emptyStatement(stmt string) bool {
	for _, b := range []byte(stmt) {
		switch b {
		case ' ', '\n', '\r', '\t', '\v', '\f':
		case ';':
		default:
			return false
		}
	}
	return true
}
