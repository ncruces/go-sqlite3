package fsutil

import (
	"io/fs"
	"testing"
)

func TestFileModeFromUnix(t *testing.T) {
	tests := []struct {
		mode fs.FileMode
		want fs.FileMode
	}{
		{0010754, 0754 | fs.ModeNamedPipe},
		{0020754, 0754 | fs.ModeCharDevice | fs.ModeDevice},
		{0040754, 0754 | fs.ModeDir},
		{0060754, 0754 | fs.ModeDevice},
		{0100754, 0754},
		{0120754, 0754 | fs.ModeSymlink},
		{0140754, 0754 | fs.ModeSocket},
		{0170754, 0754 | fs.ModeIrregular},
	}
	for _, tt := range tests {
		t.Run(tt.mode.String(), func(t *testing.T) {
			if got := FileModeFromUnix(tt.mode); got != tt.want {
				t.Errorf("fixMode() = %o, want %o", got, tt.want)
			}
		})
	}
}

func FuzzParseFileMode(f *testing.F) {
	f.Add("---------")
	f.Add("rwxrwxrwx")
	f.Add("----------")
	f.Add("-rwxrwxrwx")
	f.Add("b")
	f.Add("b---------")
	f.Add("drwxrwxrwx")
	f.Add("dalTLDpSugct?")
	f.Add("dalTLDpSugct?---------")
	f.Add("dalTLDpSugct?rwxrwxrwx")
	f.Add("dalTLDpSugct?----------")

	f.Fuzz(func(t *testing.T, str string) {
		mode, err := ParseFileMode(str)
		if err != nil {
			return
		}
		got := mode.String()
		if got != str {
			t.Errorf("was %q, got %q (%o)", str, got, mode)
		}
	})
}
