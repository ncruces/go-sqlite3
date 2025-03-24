// Package ipaddr provides functions to manipulate IPs and CIDRs.
//
// It provides the following functions:
//   - ipcontains(prefix, ip)
//   - ipoverlaps(prefix1, prefix2)
//   - ipfamily(ip/prefix)
//   - iphost(ip/prefix)
//   - ipmasklen(prefix)
//   - ipnetwork(prefix)
package ipaddr

import (
	"errors"
	"net/netip"

	"github.com/ncruces/go-sqlite3"
)

// Register IP/CIDR functions for a database connection.
func Register(db *sqlite3.Conn) error {
	const flags = sqlite3.DETERMINISTIC | sqlite3.INNOCUOUS
	return errors.Join(
		db.CreateFunction("ipcontains", 2, flags, contains),
		db.CreateFunction("ipoverlaps", 2, flags, overlaps),
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

func overlaps(ctx sqlite3.Context, arg ...sqlite3.Value) {
	prefix1, err := netip.ParsePrefix(arg[0].Text())
	if err != nil {
		ctx.ResultError(err)
		return // notest
	}
	prefix2, err := netip.ParsePrefix(arg[0].Text())
	if err != nil {
		ctx.ResultError(err)
		return // notest
	}
	ctx.ResultBool(prefix1.Overlaps(prefix2))
}

func family(ctx sqlite3.Context, arg ...sqlite3.Value) {
	addr, err := addr(arg[0].Text())
	if err != nil {
		ctx.ResultError(err)
		return // notest
	}
	switch {
	case addr.Is4():
		ctx.ResultInt(4)
	case addr.Is6():
		ctx.ResultInt(6)
	}
}

func host(ctx sqlite3.Context, arg ...sqlite3.Value) {
	addr, err := addr(arg[0].Text())
	if err != nil {
		ctx.ResultError(err)
		return // notest
	}
	buf, _ := addr.MarshalText()
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

func addr(text string) (netip.Addr, error) {
	addr, err := netip.ParseAddr(text)
	if err != nil {
		if prefix, err := netip.ParsePrefix(text); err == nil {
			return prefix.Addr(), nil
		}
		if addrpt, err := netip.ParseAddrPort(text); err == nil {
			return addrpt.Addr(), nil
		}
	}
	return addr, err
}
