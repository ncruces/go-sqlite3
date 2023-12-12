package pivot

import (
	"testing"

	"github.com/ncruces/go-sqlite3"
)

func Test_operator(t *testing.T) {
	tests := []struct {
		op   sqlite3.IndexConstraintOp
		want string
	}{
		{sqlite3.INDEX_CONSTRAINT_EQ, "="},
		{sqlite3.INDEX_CONSTRAINT_LT, "<"},
		{sqlite3.INDEX_CONSTRAINT_GT, ">"},
		{sqlite3.INDEX_CONSTRAINT_LE, "<="},
		{sqlite3.INDEX_CONSTRAINT_GE, ">="},
		{sqlite3.INDEX_CONSTRAINT_NE, "<>"},
		{sqlite3.INDEX_CONSTRAINT_IS, "IS"},
		{sqlite3.INDEX_CONSTRAINT_ISNOT, "IS NOT"},
		{sqlite3.INDEX_CONSTRAINT_REGEXP, "REGEXP"},
		{sqlite3.INDEX_CONSTRAINT_MATCH, "MATCH"},
		{sqlite3.INDEX_CONSTRAINT_GLOB, "GLOB"},
		{sqlite3.INDEX_CONSTRAINT_LIKE, "LIKE"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := operator(tt.op); got != tt.want {
				t.Errorf("operator() = %v, want %v", got, tt.want)
			}
		})
	}
}
