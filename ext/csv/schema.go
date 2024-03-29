package csv

import (
	"strconv"
	"strings"

	"github.com/ncruces/go-sqlite3"
)

func getSchema(header bool, columns int, row []string) string {
	var sep string
	var str strings.Builder
	str.WriteString("CREATE TABLE x(")

	if 0 <= columns && columns < len(row) {
		row = row[:columns]
	}
	for i, f := range row {
		str.WriteString(sep)
		if header && f != "" {
			str.WriteString(sqlite3.QuoteIdentifier(f))
		} else {
			str.WriteString("c")
			str.WriteString(strconv.Itoa(i + 1))
		}
		str.WriteString(" TEXT")
		sep = ","
	}
	for i := len(row); i < columns; i++ {
		str.WriteString(sep)
		str.WriteString("c")
		str.WriteString(strconv.Itoa(i + 1))
		str.WriteString(" TEXT")
		sep = ","
	}
	str.WriteByte(')')

	return str.String()
}
