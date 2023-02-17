package driver

type errorString string

func (e errorString) Error() string { return string(e) }

const assertErr = errorString("sqlite3: assertion failed")
