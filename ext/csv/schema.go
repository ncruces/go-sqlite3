package csv

import (
	"strconv"
	"strings"

	"github.com/ncruces/go-sqlite3"
)

func getSchema(header bool, columns int, row []string) string {
	var sep string
	var buf strings.Builder
	buf.WriteString("CREATE TABLE x(")

	if 0 <= columns && columns < len(row) {
		row = row[:columns]
	}
	for i, f := range row {
		buf.WriteString(sep)
		if header && f != "" {
			buf.WriteString(sqlite3.QuoteIdentifier(f))
		} else {
			buf.WriteString("c")
			buf.WriteString(strconv.Itoa(i + 1))
		}
		buf.WriteString(" TEXT")
		sep = ","
	}
	for i := len(row); i < columns; i++ {
		buf.WriteString(sep)
		buf.WriteString("c")
		buf.WriteString(strconv.Itoa(i + 1))
		buf.WriteString(" TEXT")
		sep = ","
	}
	buf.WriteByte(')')

	return buf.String()
}
