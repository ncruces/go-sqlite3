package parquet

import (
	"os"
	"strings"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/ncruces/go-sqlite3/util/osutil"
	"github.com/ncruces/go-sqlite3/util/sql3util"
	"github.com/parquet-go/parquet-go"
)

func Register(db *sqlite3.Conn) error {
	declare := func(db *sqlite3.Conn, _, _, _ string, arg ...string) (_ *table, err error) {
		if len(arg) == 0 {
			return nil, util.ErrorString(`parquet: must specify a filename`)
		}

		file, err := osutil.OpenFile(sql3util.Unquote(arg[0]), os.O_RDONLY, 0)
		if err != nil {
			return nil, err
		}

		reader := parquet.NewReader(file)

		column := make(map[int]string)

		var schema strings.Builder
		schema.WriteString("CREATE TABLE x(")
		for i, field := range reader.Schema().Fields() {
			if i > 0 {
				schema.WriteByte(',')
			}
			schema.WriteString(sqlite3.QuoteIdentifier(field.Name()))
			schema.WriteByte(' ')
			switch field.Type().Kind() {
			case parquet.Boolean:
				schema.WriteString("BOOLEAN")
			case parquet.Int32, parquet.Int64, parquet.Int96:
				schema.WriteString("INTEGER")
			case parquet.Float, parquet.Double:
				schema.WriteString("REAL")
			case parquet.ByteArray, parquet.FixedLenByteArray:
				schema.WriteString("TEXT")
			}
			// Save the column name
			column[i] = field.Name()
		}
		schema.WriteString(");")
		err = db.DeclareVTab(schema.String())
		if err != nil {
			return nil, err
		}
		return &table{}, nil
	}

	return sqlite3.CreateModule(db, "parquet", declare, declare)
}

type table struct {
}
