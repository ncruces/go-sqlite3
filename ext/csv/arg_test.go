package csv

import (
	"testing"

	"github.com/ncruces/go-sqlite3/util/vtabutil"
)

func Test_uintArg(t *testing.T) {
	t.Parallel()

	tests := []struct {
		arg string
		key string
		val int
		err bool
	}{
		{"columns 1", "columns 1", 0, true},
		{"columns  = 1", "columns", 1, false},
		{"columns\t= 2", "columns", 2, false},
		{" columns = 3", "columns", 3, false},
		{" columns = -1", "columns", 0, true},
		{" columns = 32768", "columns", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.arg, func(t *testing.T) {
			key, val := vtabutil.NamedArg(tt.arg)
			if key != tt.key {
				t.Errorf("NamedArg() %v, want err %v", key, tt.key)
			}
			got, err := uintArg(key, val)
			if (err != nil) != tt.err {
				t.Fatalf("uintArg() error = %v, want err %v", err, tt.err)
			}
			if got != tt.val {
				t.Errorf("uintArg() = %v, want %v", got, tt.val)
			}
		})
	}
}

func Test_boolArg(t *testing.T) {
	tests := []struct {
		arg string
		key string
		val bool
		err bool
	}{
		{"header", "header", true, false},
		{"header\t= 1", "header", true, false},
		{" header = 0", "header", false, false},
		{" header = TrUe", "header", true, false},
		{" header = FaLsE", "header", false, false},
		{" header = Yes", "header", true, false},
		{" header = nO", "header", false, false},
		{" header = On", "header", true, false},
		{" header = Off", "header", false, false},
		{" header = T", "header", false, true},
		{" header = f", "header", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.arg, func(t *testing.T) {
			key, val := vtabutil.NamedArg(tt.arg)
			if key != tt.key {
				t.Errorf("NamedArg() %v, want err %v", key, tt.key)
			}
			got, err := boolArg(key, val)
			if (err != nil) != tt.err {
				t.Fatalf("boolArg() error = %v, want err %v", err, tt.err)
			}
			if got != tt.val {
				t.Errorf("boolArg() = %v, want %v", got, tt.val)
			}
		})
	}
}

func Test_runeArg(t *testing.T) {
	tests := []struct {
		arg string
		key string
		val rune
		err bool
	}{
		{"comma", "comma", 0, true},
		{"comma\t= ,", "comma", ',', false},
		{" comma = ;", "comma", ';', false},
		{" comma = ;;", "comma", 0, true},
		{` comma = '\t`, "comma", 0, true},
		{` comma = '\t'`, "comma", '\t', false},
		{` comma = "\t"`, "comma", '\t', false},
	}
	for _, tt := range tests {
		t.Run(tt.arg, func(t *testing.T) {
			key, val := vtabutil.NamedArg(tt.arg)
			if key != tt.key {
				t.Errorf("NamedArg() %v, want err %v", key, tt.key)
			}
			got, err := runeArg(key, val)
			if (err != nil) != tt.err {
				t.Fatalf("runeArg() error = %v, want err %v", err, tt.err)
			}
			if got != tt.val {
				t.Errorf("runeArg() = %v, want %v", got, tt.val)
			}
		})
	}
}
