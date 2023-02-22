package sqlite3

// ZeroBlob represents a zero-filled, length n BLOB
// that can be used as an argument to
// [database/sql.DB.Exec] and similar methods.
type ZeroBlob int64
