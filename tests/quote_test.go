package tests

import (
	"database/sql"
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/ncruces/go-sqlite3"
)

func TestQuote(t *testing.T) {
	tests := []struct {
		val  any
		want string
	}{
		{`abc`, "'abc'"},
		{`a"bc`, "'a\"bc'"},
		{`a'bc`, "'a''bc'"},
		{"\x07bc", "'\abc'"},
		{"\x1c\n", "'\x1c\n'"},
		{"\xB0\x00\x0B", "'\xB0'"},
		{[]byte("\xB0\x00\x0B"), "x'B0000B'"},

		{0, "0"},
		{true, "1"},
		{false, "0"},
		{nil, "NULL"},
		{math.NaN(), "NULL"},
		{math.Inf(1), "9.0e999"},
		{math.Inf(-1), "-9.0e999"},
		{math.Pi, "3.141592653589793"},
		{int64(math.MaxInt64), "9223372036854775807"},
		{time.Unix(0, 0).UTC(), "'1970-01-01T00:00:00Z'"},
		{sqlite3.ZeroBlob(4), "x'00000000'"},
		{int8(0), "0"},
		{uint(0), "0"},
		{float32(0), "0"},
		{(*string)(nil), "NULL"},
		{json.Number("0"), "'0'"},
		{&sql.RawBytes{'0'}, "x'30'"},
		{t, ""}, // panic
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil && tt.want != "" {
					t.Errorf("Quote(%q) = %v", tt.val, r)
				}
			}()

			if got := sqlite3.Quote(tt.val); got != tt.want {
				t.Errorf("Quote(%v) = %q, want %q", tt.val, got, tt.want)
			}
		})
	}
}

func TestQuoteIdentifier(t *testing.T) {
	tests := []struct {
		id   string
		want string
	}{
		{`abc`, `"abc"`},
		{`a"bc`, `"a""bc"`},
		{`a'bc`, `"a'bc"`},
		{"\x07bc", "\"\abc\""},
		{"\x1c\n", "\"\x1c\n\""},
		{"\xB0\x00\x0B", ""}, // panic
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil && tt.want != "" {
					t.Errorf("QuoteIdentifier(%q) = %v", tt.id, r)
				}
			}()

			if got := sqlite3.QuoteIdentifier(tt.id); got != tt.want {
				t.Errorf("QuoteIdentifier(%v) = %q, want %q", tt.id, got, tt.want)
			}
		})
	}
}
