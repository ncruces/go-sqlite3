package parser

import (
	"bytes"
	"unicode"
)

type LibC struct{}

func (LibC) Istrlen(m *Module, v0 int32) int32 {
	return int32(bytes.IndexByte(m.Memory[v0:], 0))
}

func (LibC) Itolower(m *Module, v0 int32) int32 {
	if v0 > unicode.MaxASCII {
		return v0
	}
	return unicode.ToLower(v0)
}
