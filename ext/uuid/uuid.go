// Package uuid provides functions to generate RFC 4122 UUIDs.
//
// https://sqlite.org/src/file/ext/misc/uuid.c
package uuid

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/internal/util"
)

// Register registers the SQL functions:
//
//   - uuid([ version [, domain/namespace, [ id/data ]]]):
//     to generate a UUID as a string
//   - uuid_str(u):
//     to convert a UUID into a well-formed UUID string
//   - uuid_blob(u):
//     to convert a UUID into a 16-byte blob
//   - uuid_extract_version(u):
//     to extract the version of a RFC 4122 UUID
//   - uuid_extract_timestamp(u):
//     to extract the timestamp of a version 1/2/6/7 UUID
//   - gen_random_uuid(u):
//     to generate a version 4 (random) UUID
func Register(db *sqlite3.Conn) error {
	const flags = sqlite3.DETERMINISTIC | sqlite3.INNOCUOUS
	return errors.Join(
		db.CreateFunction("uuid", 0, sqlite3.INNOCUOUS, generate),
		db.CreateFunction("uuid", 1, sqlite3.INNOCUOUS, generate),
		db.CreateFunction("uuid", 2, sqlite3.INNOCUOUS, generate),
		db.CreateFunction("uuid", 3, sqlite3.INNOCUOUS, generate),
		db.CreateFunction("uuid_str", 1, flags, toString),
		db.CreateFunction("uuid_blob", 1, flags, toBlob),
		db.CreateFunction("uuid_extract_version", 1, flags, version),
		db.CreateFunction("uuid_extract_timestamp", 1, flags, timestamp),
		db.CreateFunction("gen_random_uuid", 0, sqlite3.INNOCUOUS, generate))
}

func generate(ctx sqlite3.Context, arg ...sqlite3.Value) {
	var (
		ver int
		err error
		u   uuid.UUID
	)

	if len(arg) > 0 {
		ver = arg[0].Int()
	} else {
		ver = 4
	}

	switch ver {
	case 1:
		u, err = uuid.NewUUID()
	case 4:
		u, err = uuid.NewRandom()
	case 6:
		u, err = uuid.NewV6()
	case 7:
		u, err = uuid.NewV7()

	case 2:
		var domain uuid.Domain
		if len(arg) > 1 {
			domain = uuid.Domain(arg[1].Int64())
			if domain == 0 {
				if txt := arg[1].RawText(); len(txt) > 0 {
					switch txt[0] | 0x20 { // to lower
					case 'g': // group
						domain = uuid.Group
					case 'o': // org
						domain = uuid.Org
					}
				}
			}
		}
		switch {
		case len(arg) > 2:
			u, err = uuid.NewDCESecurity(domain, uint32(arg[2].Int64()))
		case domain == uuid.Person:
			u, err = uuid.NewDCEPerson()
		case domain == uuid.Group:
			u, err = uuid.NewDCEGroup()
		default:
			err = util.ErrorString("missing id")
		}

	case 3, 5:
		if len(arg) < 2 {
			err = util.ErrorString("missing data")
			break
		}
		ns, err := fromValue(arg[1])
		if err != nil {
			space := arg[1].RawText()
			switch {
			case bytes.EqualFold(space, []byte("url")):
				ns = uuid.NameSpaceURL
			case bytes.EqualFold(space, []byte("oid")):
				ns = uuid.NameSpaceOID
			case bytes.EqualFold(space, []byte("dns")):
				ns = uuid.NameSpaceDNS
			case bytes.EqualFold(space, []byte("fqdn")):
				ns = uuid.NameSpaceDNS
			case bytes.EqualFold(space, []byte("x500")):
				ns = uuid.NameSpaceX500
			default:
				ctx.ResultError(err)
				return // notest
			}
		}
		if ver == 3 {
			u = uuid.NewMD5(ns, arg[2].RawBlob())
		} else {
			u = uuid.NewSHA1(ns, arg[2].RawBlob())
		}

	default:
		err = fmt.Errorf("invalid version: %d", ver)
	}

	if err != nil {
		ctx.ResultError(fmt.Errorf("uuid: %w", err)) // notest
	} else {
		ctx.ResultText(u.String())
	}
}

func fromValue(arg sqlite3.Value) (u uuid.UUID, err error) {
	switch t := arg.Type(); t {
	case sqlite3.TEXT:
		u, err = uuid.ParseBytes(arg.RawText())
		if err != nil {
			err = fmt.Errorf("uuid: %w", err)
		}

	case sqlite3.BLOB:
		blob := arg.RawBlob()
		if len := len(blob); len != 16 {
			err = fmt.Errorf("uuid: invalid BLOB length: %d", len)
		} else {
			copy(u[:], blob)
		}

	default:
		err = fmt.Errorf("uuid: invalid type: %v", t)
	}
	return u, err
}

func toBlob(ctx sqlite3.Context, arg ...sqlite3.Value) {
	u, err := fromValue(arg[0])
	if err != nil {
		ctx.ResultError(err) // notest
	} else {
		ctx.ResultBlob(u[:])
	}
}

func toString(ctx sqlite3.Context, arg ...sqlite3.Value) {
	u, err := fromValue(arg[0])
	if err != nil {
		ctx.ResultError(err) // notest
	} else {
		ctx.ResultText(u.String())
	}
}

func version(ctx sqlite3.Context, arg ...sqlite3.Value) {
	u, err := fromValue(arg[0])
	if err != nil {
		ctx.ResultError(err)
		return // notest
	}
	if u.Variant() == uuid.RFC4122 {
		ctx.ResultInt64(int64(u.Version()))
	}
}

func timestamp(ctx sqlite3.Context, arg ...sqlite3.Value) {
	u, err := fromValue(arg[0])
	if err != nil {
		ctx.ResultError(err)
		return // notest
	}
	if u.Variant() == uuid.RFC4122 {
		switch u.Version() {
		case 1, 2, 6, 7:
			ctx.ResultTime(
				time.Unix(u.Time().UnixTime()),
				sqlite3.TimeFormatDefault)
		}
	}
}
