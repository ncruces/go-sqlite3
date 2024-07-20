package util

import "testing"

func TestErrorJoiner(t *testing.T) {
	var errs ErrorJoiner
	errs.Join(NilErr, OOMErr)
	for i, e := range []error{NilErr, OOMErr} {
		if e != errs[i] {
			t.Fail()
		}
	}
}
