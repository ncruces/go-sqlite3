package sqlite3

import "github.com/ncruces/go-sqlite3/internal/util"

type Bulk struct {
	c      *Conn
	prefix string
	suffix string
	buffer []byte
	bufptr uint32
}

const _BULK_SIZE = 1024 * 1024

func (c *Conn) CreateBulk(prefix, suffix string) (*Bulk, error) {
	if len(prefix)+len(suffix) > _BULK_SIZE/2 {
		return nil, TOOBIG
	}
	ptr := c.new(_BULK_SIZE)
	buf := util.View(c.mod, ptr, _BULK_SIZE)
	copy(buf, prefix)
	buf = buf[len(prefix):len(prefix)]
	return &Bulk{
		c:      c,
		prefix: prefix,
		suffix: suffix,
		buffer: buf,
		bufptr: ptr,
	}, nil
}

func (b *Bulk) Close() error {
	if b == nil || b.c == nil {
		return nil
	}

	err := b.Flush()
	b.c.free(b.bufptr)
	b.c = nil
	return err
}

func (b *Bulk) Flush() error {
	if len(b.buffer) == 0 {
		return nil
	}
	if cap(b.buffer)-len(b.buffer) <= len(b.suffix) {
		return TOOBIG
	}
	b.c.checkInterrupt(b.c.handle)
	b.buffer = append(b.buffer, b.suffix...)
	b.buffer = append(b.buffer, 0)
	b.buffer = b.buffer[:0]
	r := b.c.call("sqlite3_exec", uint64(b.c.handle), uint64(b.bufptr), 0, 0, 0)
	return b.c.error(r)
}

func (b *Bulk) AppendRow(args ...any) error {
	buf := b.buffer

	if off := len(buf); off != 0 {
		buf = append(buf[off:], ',')
	}

	buf = append(buf, '(')
	for i, arg := range args {
		if i != 0 {
			buf = append(buf, ',')
		}
		buf = append(buf, Quote(arg)...)
	}
	buf = append(buf, ')')

	if len(buf)+len(b.suffix) >= cap(b.buffer)-len(b.buffer) {
		if err := b.Flush(); err != nil {
			return err
		}
		if buf[0] == ',' {
			buf = buf[1:]
		}
	}
	if len(buf)+len(b.suffix) >= cap(b.buffer)-len(b.buffer) {
		return TOOBIG
	}

	b.buffer = append(b.buffer, buf...)
	return nil
}
