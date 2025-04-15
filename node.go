package chainor

import (
	"log"
	"runtime/debug"

	"github.com/panjf2000/ants/v2"
)

const (
	DefaultMaxBlockingTasks = 20
)

type (
	node struct {
		opt        *option
		calls      TaskFuncs
		workerpool *ants.Pool

		priority int
	}
)

func (n *node) panicHandler() func(interface{}) {
	return func(err interface{}) {
		log.Printf("executor coroutine panic: %v\n%v", err, string(debug.Stack()))
		n.clean()
	}
}

func (n *node) clean() {
}

func (n *node) concrete() *node {
	{
		switch n.opt.cct.mode() {
		case workerpoolM:
			if wp, err := ants.NewPool(n.opt.cct.count(),
				ants.WithPanicHandler(n.panicHandler()),
				ants.WithMaxBlockingTasks(DefaultMaxBlockingTasks)); err != nil {
				log.Fatalf("failed to new task pool: %v\n", err)
			} else {
				n.workerpool = wp
			}
		}
	}
	{
		if len(n.opt.funcs) > 0 {
			n.calls = append(n.calls, n.opt.funcs...)
		}
	}
	return n
}
