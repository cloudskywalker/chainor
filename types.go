package chainor

import (
	"errors"

	"go.linecorp.com/garr/queue"
)

type (
	M map[string]any

	Chainor struct {
		opt   *option
		queue queue.Queue
	}

	// TaskFunc 任务函数，lastResult 为上一个任务的返回值，之所以为数组形式，是因为可能会有多个并行任务的返回值
	TaskFunc func(ctx *TaskContext, lastResult []any) (result any, err error)

	TaskFuncs []TaskFunc
)

// 内置的优先级常量
const (
	P_Min    int = -200
	P_Low        = -100
	P_Normal     = 0
	P_High       = 100
	P_Max        = 200
)

var (
	// ErrTimeout 超时错误
	ErrTimeout = errors.New("E_CHAINOR_TIMEOUT")

	// ErrNoPassed 没有任务函数命中，例如当指定 WithAnyPassed 时，没有一个任务的返回值符合预期，则会返回该错误
	ErrNoPassed = errors.New("E_CHAINOR_NO_PASSED")

	errNotDone = errors.New("not done")
)
