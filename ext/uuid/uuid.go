package uuid

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/ncruces/go-sqlite3"
)

func Register(db *sqlite3.Conn) {
	flags := sqlite3.DETERMINISTIC | sqlite3.INNOCUOUS
	db.CreateFunction("uuid", 0, sqlite3.INNOCUOUS, generate)
	db.CreateFunction("uuid", 1, sqlite3.INNOCUOUS, generate)
	db.CreateFunction("uuid_str", 1, flags, toString)
	db.CreateFunction("uuid_blob", 1, flags, toBLOB)
}

func generate(ctx sqlite3.Context, arg ...sqlite3.Value) {
	var (
		version int
		err     error
		u       uuid.UUID
	)

	if len(arg) > 0 {
		version = arg[0].Int()
	}

	switch version {
	case 0, 4:
		u, err = uuid.NewRandom()
	case 1:
		u, err = uuid.NewUUID()
	case 6:
		u, err = uuid.NewV6()
	case 7:
		u, err = uuid.NewV7()
	default:
		err = fmt.Errorf("invalid version: %d", version)
	}

	if err != nil {
		ctx.ResultError(fmt.Errorf("uuid: %w", err))
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

func toBLOB(ctx sqlite3.Context, arg ...sqlite3.Value) {
	u, err := fromValue(arg[0])
	if err != nil {
		ctx.ResultError(err)
	} else {
		ctx.ResultBlob(u[:])
	}
}

func toString(ctx sqlite3.Context, arg ...sqlite3.Value) {
	u, err := fromValue(arg[0])
	if err != nil {
		ctx.ResultError(err)
	} else {
		ctx.ResultText(u.String())
	}
}
