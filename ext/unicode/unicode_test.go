package unicode

import "testing"

func Test_like2regex(t *testing.T) {
	tests := []struct {
		pattern string
		escape  rune
		want    string
	}{
		{`a`, -1, `(?is)a`},
		{`a.`, -1, `(?is)a\.`},
		{`a%`, -1, `(?is)a.*`},
		{`a\`, -1, `(?is)a\\`},
		{`a_b`, -1, `(?is)a.b`},
		{`a|b`, '|', `(?is)ab`},
		{`a|_`, '|', `(?is)a_`},
	}
	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			if got := like2regex(tt.pattern, tt.escape); got != tt.want {
				t.Errorf("like2regex() = %v, want %v", got, tt.want)
			}
		})
	}
}
