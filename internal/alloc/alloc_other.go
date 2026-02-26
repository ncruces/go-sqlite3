//go:build !unix && !windows

package alloc

type Memory struct {
	Max int32
}

func (m Memory) Grow(mem *[]byte, delta, _ int32) int32 {
	buf := *mem
	len := len(buf)
	old := int32(len >> 16)
	if delta == 0 {
		return old
	}
	new := old + delta
	add := int(new)<<16 - len
	if new > m.Max || add < 0 {
		return -1
	}
	*mem = append(buf, make([]byte, add)...)
	return old
}

func (m Memory) Free() {}
