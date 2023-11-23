package csv

import "testing"

func Test_uintParam(t *testing.T) {
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
			key, val := getParam(tt.arg)
			if key != tt.key {
				t.Errorf("getParam() %v, want err %v", key, tt.key)
			}
			got, err := uintParam(key, val)
			if (err != nil) != tt.err {
				t.Fatalf("uintParam() error = %v, want err %v", err, tt.err)
			}
			if got != tt.val {
				t.Errorf("uintParam() = %v, want %v", got, tt.val)
			}
		})
	}
}

func Test_boolParam(t *testing.T) {
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
			key, val := getParam(tt.arg)
			if key != tt.key {
				t.Errorf("getParam() %v, want err %v", key, tt.key)
			}
			got, err := boolParam(key, val)
			if (err != nil) != tt.err {
				t.Fatalf("boolParam() error = %v, want err %v", err, tt.err)
			}
			if got != tt.val {
				t.Errorf("boolParam() = %v, want %v", got, tt.val)
			}
		})
	}
}

func Test_runeParam(t *testing.T) {
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
			key, val := getParam(tt.arg)
			if key != tt.key {
				t.Errorf("getParam() %v, want err %v", key, tt.key)
			}
			got, err := runeParam(key, val)
			if (err != nil) != tt.err {
				t.Fatalf("runeParam() error = %v, want err %v", err, tt.err)
			}
			if got != tt.val {
				t.Errorf("runeParam() = %v, want %v", got, tt.val)
			}
		})
	}
}
