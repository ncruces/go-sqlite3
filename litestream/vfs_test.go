package litestream

import (
	"slices"
	"strconv"
	"testing"

	"github.com/benbjohnson/litestream"

	_ "github.com/ncruces/go-sqlite3/embed"
)

func Test_pollLevels(t *testing.T) {
	tests := []struct {
		minLevel int
		want     []int
	}{
		{minLevel: -1, want: []int{0, 1, litestream.SnapshotLevel}},
		{minLevel: 0, want: []int{0, 1, litestream.SnapshotLevel}},
		{minLevel: 1, want: []int{1, litestream.SnapshotLevel}},
		{minLevel: 2, want: []int{2, litestream.SnapshotLevel}},
		{minLevel: 3, want: []int{3, litestream.SnapshotLevel}},
		{minLevel: litestream.SnapshotLevel, want: []int{litestream.SnapshotLevel}},
		{minLevel: litestream.SnapshotLevel + 1, want: []int{litestream.SnapshotLevel}},
	}
	for _, tt := range tests {
		t.Run(strconv.Itoa(tt.minLevel), func(t *testing.T) {
			got := pollLevels(tt.minLevel)
			if !slices.Equal(got, tt.want) {
				t.Errorf("pollLevels() = %v, want %v", got, tt.want)
			}
		})
	}
}
