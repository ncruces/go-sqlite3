package ipaddr

import (
	"errors"
	"net/netip"

	"github.com/ncruces/go-sqlite3"
)

func Register(db *sqlite3.Conn) error {
	const flags = sqlite3.DETERMINISTIC | sqlite3.INNOCUOUS
	return errors.Join(
		db.CreateFunction("ipcontains", 2, flags, contains),
		db.CreateFunction("ipfamily", 1, flags, family),
		db.CreateFunction("iphost", 1, flags, host),
		db.CreateFunction("ipmasklen", 1, flags, masklen),
		db.CreateFunction("ipnetwork", 1, flags, network))
}

func contains(ctx sqlite3.Context, arg ...sqlite3.Value) {
	prefix, err := netip.ParsePrefix(arg[0].Text())
	if err != nil {
		ctx.ResultError(err)
		return // notest
	}
	addr, err := netip.ParseAddr(arg[1].Text())
	if err != nil {
		ctx.ResultError(err)
		return // notest
	}
	ctx.ResultBool(prefix.Contains(addr))
}

func family(ctx sqlite3.Context, arg ...sqlite3.Value) {
	text := arg[0].Text()
	addr, err := netip.ParseAddr(text)
	if err != nil {
		if prefix, err := netip.ParsePrefix(text); err == nil {
			addr = prefix.Addr()
		} else {
			ctx.ResultError(err)
			return // notest
		}
	}
	switch {
	case addr.Is4():
		ctx.ResultInt(4)
	case addr.Is6():
		ctx.ResultInt(6)
	}
}

func host(ctx sqlite3.Context, arg ...sqlite3.Value) {
	prefix, err := netip.ParsePrefix(arg[0].Text())
	if err != nil {
		ctx.ResultError(err)
		return // notest
	}
	buf, _ := prefix.Addr().MarshalText()
	ctx.ResultRawText(buf)
}

func masklen(ctx sqlite3.Context, arg ...sqlite3.Value) {
	prefix, err := netip.ParsePrefix(arg[0].Text())
	if err != nil {
		ctx.ResultError(err)
		return // notest
	}
	ctx.ResultInt(prefix.Bits())
}

func network(ctx sqlite3.Context, arg ...sqlite3.Value) {
	prefix, err := netip.ParsePrefix(arg[0].Text())
	if err != nil {
		ctx.ResultError(err)
		return // notest
	}
	buf, _ := prefix.Masked().MarshalText()
	ctx.ResultRawText(buf)
}
