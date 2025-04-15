package chainor

import (
	"time"
)

type (
	concurrent uint32

	option struct {
		name    string
		param   any
		props   M
		timeout time.Duration

		funcs []TaskFunc

		skippedFunc   func([]any) bool
		anyPassedFunc func(any) bool

		priority int
		cct      concurrent

		skipResult bool
	}
	optionable interface {
		call(*option)
	}

	Option        func(opt *option)
	TaskOption    func(opt *option)
	ChainorOption func(opt *option)
)

const (
	parallelM uint16 = iota + 1
	workerpoolM
)

func newConcurrent(mode uint16, count uint16) concurrent {
	return concurrent(mode)<<16 | concurrent(count)
}

func (c concurrent) mode() uint16 {
	return uint16(c >> 16)
}

func (c concurrent) count() int {
	return int(c << 16 >> 16)
}

func (o Option) call(opt *option) {
	o(opt)
}

func (o TaskOption) call(opt *option) {
	o(opt)
}

func (o ChainorOption) call(opt *option) {
	o(opt)
}

func mergeOption[T optionable](withFunc ...T) *option {
	o := &option{}

	for i := range withFunc {
		withFunc[i].call(o)
	}
	return o
}

// WithParam Invoke 参数
func WithParam(param any) Option {
	return func(opt *option) {
		opt.param = param
	}
}

// WithProps Invoke 属性
func WithProps(props M) Option {
	return func(opt *option) {
		opt.props = props
	}
}

// WithTimeout Invoke 超时时间，超时时返回内置错误 ErrTimeout
func WithTimeout(timeout time.Duration) Option {
	return func(opt *option) {
		opt.timeout = timeout
	}
}

func withSkipReuslt() TaskOption {
	return func(opt *option) {
		opt.skipResult = true
	}
}

// WithTaskParam 节点任务参数
func WithTaskParam(param any) TaskOption {
	return func(opt *option) {
		opt.param = param
	}
}

// WithTaskProps 节点任务属性
func WithTaskProps(props M) TaskOption {
	return func(opt *option) {
		opt.props = props
	}
}

// WithParallelFunc 并行任务函数，可配置多个并行函数
func WithParallelFunc(tasks ...TaskFunc) TaskOption {
	return func(opt *option) {
		opt.funcs = tasks
	}
}

// WithSkipped 符合指定条件时跳过该任务节点
func WithSkipped(predicate func(result []any) bool) TaskOption {
	return func(opt *option) {
		opt.skippedFunc = predicate
	}
}

// WithAnyPassed 任意一个任务函数执行通过则该节点任务完成，若 predicates 没有时，则默认 TaskFunc 执行成功为通过
func WithAnyPassed(predicates ...func(result any) bool) TaskOption {
	predicateFunc := func(_ any) bool {
		return true
	}
	if len(predicates) != 0 {
		predicateFunc = predicates[0]
	}
	return func(opt *option) {
		opt.anyPassedFunc = predicateFunc
	}
}

// WithParallel 并行模式
//
// parallel 指定并行度，当与 WithParallelFunc 结合使用时，每个并行函数都会有相同的并行度
func WithParallel(parallel uint16) TaskOption {
	return func(opt *option) {
		opt.cct = newConcurrent(parallelM, parallel)
	}
}

// WithWorkerPool 协程池模式，与 WithParallel 互斥
//
// pool 指定协程池容量，与 WithParallelFunc 结合使用时，所有任务函数共享同一份协程池
func WithWorkerPool(pool uint16) TaskOption {
	return func(opt *option) {
		opt.cct = newConcurrent(workerpoolM, pool)
	}
}

// WithChainorName Chainor 名称
func WithChainorName(name string) ChainorOption {
	return func(opt *option) {
		opt.name = name
	}
}

// WithChainorParam Chainor 参数
func WithChainorParam(param any) ChainorOption {
	return func(opt *option) {
		opt.param = param
	}
}

// WithChainorProps Chainor 属性
func WithChainorProps(props M) ChainorOption {
	return func(opt *option) {
		opt.props = props
	}
}
