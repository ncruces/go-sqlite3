package sql3util

import (
	"errors"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/internal/sqlite3_wrap"
)

func CodeToError(code int32) error {
	switch {
	case code == sqlite3_wrap.OK:
		return nil
	case code <= sqlite3_wrap.DONE:
		return sqlite3.ErrorCode(code)
	default:
		return sqlite3.ExtendedErrorCode(code)
	}
}

func ErrorToCode(err error) int32 {
	switch code := err.(type) {
	case nil:
		return sqlite3_wrap.OK
	case sqlite3.ErrorCode:
		return int32(code)
	case sqlite3.ExtendedErrorCode:
		return int32(code)
	case *sqlite3.Error:
		return int32(code.Code())
	}

	var xcode sqlite3.ExtendedErrorCode
	if errors.As(err, &xcode) {
		return int32(xcode)
	}
	return sqlite3_wrap.ERROR
}
