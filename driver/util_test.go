package driver

import (
	"database/sql/driver"
	"reflect"
	"testing"
)

func Test_namedValues(t *testing.T) {
	want := []driver.NamedValue{
		{Ordinal: 1, Value: true},
		{Ordinal: 2, Value: false},
	}
	got := namedValues([]driver.Value{true, false})
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
