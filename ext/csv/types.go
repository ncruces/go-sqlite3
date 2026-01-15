package csv

import "github.com/ncruces/go-sqlite3/util/sql3util"

func getColumnAffinities(schema string) ([]sql3util.Affinity, error) {
	tab, err := sql3util.ParseTable(schema)
	if err != nil {
		return nil, err
	}

	columns := tab.Columns
	types := make([]sql3util.Affinity, len(columns))
	for i, col := range columns {
		types[i] = sql3util.GetAffinity(col.Type)
	}
	return types, nil
}
