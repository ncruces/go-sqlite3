package sqlite3

import (
	"strconv"
	"strings"
)

type Error struct {
	Code         ErrorCode
	ExtendedCode ExtendedErrorCode
	str          string
	msg          string
}

func (e Error) Error() string {
	var b strings.Builder
	b.WriteString("sqlite3: ")

	if e.str != "" {
		b.WriteString(e.str)
	} else {
		b.WriteString(strconv.Itoa(int(e.Code)))
	}

	if e.msg != "" {
		b.WriteByte(':')
		b.WriteByte(' ')
		b.WriteString(e.msg)
	}

	return b.String()
}
