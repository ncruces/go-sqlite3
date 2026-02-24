package parser

import (
	"bytes"
	"unicode"
)

type LibC struct{ mod *Module }

func (l *LibC) Init(m *Module) { l.mod = m }

func (l *LibC) Xstrlen(v0 int32) int32 {
	return int32(bytes.IndexByte(l.mod.Memory[v0:], 0))
}

func (LibC) Xtolower(v0 int32) int32 {
	if v0 > unicode.MaxASCII {
		return v0
	}
	return unicode.ToLower(v0)
}
