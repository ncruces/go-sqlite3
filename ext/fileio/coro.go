//go:build !(go1.23 || goexperiment.rangefunc) || vet

package fileio

import (
	"fmt"

	"github.com/ncruces/go-sqlite3/internal/util"
)

// Adapted from: https://research.swtch.com/coro

const errCoroCanceled = util.ErrorString("coroutine canceled")

func coroNew[In, Out any](f func(In, func(Out) In) Out) (resume func(In) (Out, bool), cancel func()) {
	type msg[T any] struct {
		panic any
		val   T
	}

	cin := make(chan msg[In])
	cout := make(chan msg[Out])
	running := true
	resume = func(in In) (out Out, ok bool) {
		if !running {
			return
		}
		cin <- msg[In]{val: in}
		m := <-cout
		if m.panic != nil {
			panic(m.panic)
		}
		return m.val, running
	}
	cancel = func() {
		if !running {
			return
		}
		e := fmt.Errorf("%w", errCoroCanceled)
		cin <- msg[In]{panic: e}
		m := <-cout
		if m.panic != nil && m.panic != e {
			panic(m.panic)
		}
	}
	yield := func(out Out) In {
		cout <- msg[Out]{val: out}
		m := <-cin
		if m.panic != nil {
			panic(m.panic)
		}
		return m.val
	}
	go func() {
		defer func() {
			if running {
				running = false
				cout <- msg[Out]{panic: recover()}
			}
		}()
		var out Out
		m := <-cin
		if m.panic == nil {
			out = f(m.val, yield)
		}
		running = false
		cout <- msg[Out]{val: out}
	}()
	return resume, cancel
}
