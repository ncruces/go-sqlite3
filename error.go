package sqlite3

import (
	"runtime"
	"strconv"
	"strings"
)

// Error wraps an SQLite Error Code.
//
// https://www.sqlite.org/c3ref/errcode.html
type Error struct {
	code uint64
	str  string
	msg  string
}

// Code returns the primary error code for this error.
//
// https://www.sqlite.org/rescode.html
func (e *Error) Code() ErrorCode {
	return ErrorCode(e.code)
}

// ExtendedCode returns the extended error code for this error.
//
// https://www.sqlite.org/rescode.html
func (e *Error) ExtendedCode() ExtendedErrorCode {
	return ExtendedErrorCode(e.code)
}

// Error implements the error interface.
func (e *Error) Error() string {
	var b strings.Builder
	b.WriteString("sqlite3: ")

	if e.str != "" {
		b.WriteString(e.str)
	} else {
		b.WriteString(strconv.Itoa(int(e.code)))
	}

	if e.msg != "" {
		b.WriteByte(':')
		b.WriteByte(' ')
		b.WriteString(e.msg)
	}

	return b.String()
}

type errorString string

func (e errorString) Error() string { return string(e) }

const (
	nilErr      = errorString("sqlite3: invalid memory address or null pointer dereference")
	oomErr      = errorString("sqlite3: out of memory")
	rangeErr    = errorString("sqlite3: index out of range")
	noNulErr    = errorString("sqlite3: missing NUL terminator")
	noGlobalErr = errorString("sqlite3: could not find global: ")
	noFuncErr   = errorString("sqlite3: could not find function: ")
)

func assertErr() errorString {
	msg := "sqlite3: assertion failed"
	if _, file, line, ok := runtime.Caller(1); ok {
		msg += " (" + file + ":" + strconv.Itoa(line) + ")"
	}
	return errorString(msg)
}
