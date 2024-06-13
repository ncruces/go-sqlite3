package csv

import (
	"strings"

	"github.com/rqlite/sql"
)

type affinity byte

const (
	blob    affinity = 0
	text    affinity = 1
	numeric affinity = 2
	integer affinity = 3
	real    affinity = 4
)

func getColumnAffinities(schema string) []affinity {
	stmt, _ := sql.NewParser(strings.NewReader(schema)).ParseStatement()
	create := stmt.(*sql.CreateTableStatement)
	typs := make([]affinity, len(create.Columns))
	for i := range typs {
		var name string
		if typ := create.Columns[i].Type; typ != nil {
			name = typ.Name.Name
		}
		typs[i] = getAffinity(name)
	}
	return typs
}

func getAffinity(declType string) affinity {
	// https://sqlite.org/datatype3.html#determination_of_column_affinity
	if declType == "" {
		return blob
	}
	name := strings.ToUpper(declType)
	if strings.Contains(name, "INT") {
		return integer
	}
	if strings.Contains(name, "CHAR") || strings.Contains(name, "CLOB") || strings.Contains(name, "TEXT") {
		return text
	}
	if strings.Contains(name, "BLOB") {
		return blob
	}
	if strings.Contains(name, "REAL") || strings.Contains(name, "FLOA") || strings.Contains(name, "DOUB") {
		return real
	}
	return numeric
}
