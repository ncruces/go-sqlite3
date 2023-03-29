package util

import (
	"fmt"
	"runtime"
	"strconv"
)

type ErrorString string

func (e ErrorString) Error() string { return string(e) }

const (
	NilErr       = ErrorString("sqlite3: invalid memory address or null pointer dereference")
	OOMErr       = ErrorString("sqlite3: out of memory")
	RangeErr     = ErrorString("sqlite3: index out of range")
	NoNulErr     = ErrorString("sqlite3: missing NUL terminator")
	NoGlobalErr  = ErrorString("sqlite3: could not find global: ")
	NoFuncErr    = ErrorString("sqlite3: could not find function: ")
	BinaryErr    = ErrorString("sqlite3: no SQLite binary embed/set/loaded")
	TimeErr      = ErrorString("sqlite3: invalid time value")
	WhenceErr    = ErrorString("sqlite3: invalid whence")
	OffsetErr    = ErrorString("sqlite3: invalid offset")
	TailErr      = ErrorString("sqlite3: multiple statements")
	IsolationErr = ErrorString("sqlite3: unsupported isolation level")
)

func AssertErr() ErrorString {
	msg := "sqlite3: assertion failed"
	if _, file, line, ok := runtime.Caller(1); ok {
		msg += " (" + file + ":" + strconv.Itoa(line) + ")"
	}
	return ErrorString(msg)
}

func Finalizer[T any](skip int) func(*T) {
	msg := fmt.Sprintf("sqlite3: %T not closed", new(T))
	if _, file, line, ok := runtime.Caller(skip + 1); ok && skip >= 0 {
		msg += " (" + file + ":" + strconv.Itoa(line) + ")"
	}
	return func(*T) { panic(ErrorString(msg)) }
}
