package sqlite3

import (
	"strconv"
	"strings"
)

type Error struct {
	Code         int
	ExtendedCode int
	str          string
	msg          string
}

func (e Error) Error() string {
	var b strings.Builder
	b.WriteString("sqlite3: ")

	if e.str != "" {
		b.WriteString(e.str)
	} else {
		b.WriteString(strconv.Itoa(e.Code))
	}

	if e.msg != "" {
		b.WriteByte(':')
		b.WriteByte(' ')
		b.WriteString(e.msg)
	}

	return b.String()
}
