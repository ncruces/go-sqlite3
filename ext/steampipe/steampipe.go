package steampipe

import (
	"strings"

	"github.com/ncruces/go-sqlite3"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
)

func Register(db *sqlite3.Conn, plugin *plugin.Plugin) error {
	for name := range plugin.TableMap {
		err := sqlite3.CreateModule[*table](db, name, nil,
			func(db *sqlite3.Conn, _, _, name string, _ ...string) (*table, error) {
				table := table{
					Table: plugin.TableMap[name],
				}

				var sep string
				var str strings.Builder
				str.WriteString("CREATE TABLE x(")

				for _, col := range table.Columns {
					str.WriteString(sep)
					str.WriteString(sqlite3.QuoteIdentifier(col.Name))
					str.WriteString(" ")
					str.WriteString(col.Type.String())
					sep = ","
				}

				str.WriteByte(')')

				err := db.DeclareVTab(str.String())
				return &table, err
			})
		if err != nil {
			return err
		}
	}
	return nil
}

type table struct {
	*plugin.Table
}

func (*table) BestIndex(idx *sqlite3.IndexInfo) error {
	return sqlite3.ERROR
}

func (*table) Open() (sqlite3.VTabCursor, error) {
	return nil, nil
}
