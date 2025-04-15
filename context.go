package chainor

import (
	"context"
	"time"

	cmp "github.com/orcaman/concurrent-map/v2"
)

type (
	chainorContext struct {
		ctx    context.Context
		cancel context.CancelFunc

		f         *future
		keyValues cmp.ConcurrentMap[any]
	}

	TaskContext struct {
		s *step
	}
)

func (f *future) newChainorContext() *chainorContext {
	ctx := &chainorContext{
		ctx:       context.Background(),
		f:         f,
		keyValues: cmp.New[any](),
	}

	if f.opt.timeout != time.Duration(0) {
		ctx.ctx, ctx.cancel = context.WithTimeout(context.Background(), f.opt.timeout)
	}
	return ctx
}

func (s *step) newTaskContext() *TaskContext {
	return &TaskContext{
		s: s,
	}
}

func (c *TaskContext) Context() context.Context {
	return c.s.f.ctx.ctx
}

func (c *TaskContext) Cancel() context.CancelFunc {
	return c.s.f.ctx.cancel
}

// WithValue 全局有效，且支持并行操作
func (c *TaskContext) WithValue(key string, value any) {
	c.s.f.ctx.keyValues.Set(key, value)
}

// Value 获取 key 对应的 value
func (c *TaskContext) Value(key string) any {
	if v, ok := c.s.f.ctx.keyValues.Get(key); ok {
		return v
	}
	return nil
}

func (c *TaskContext) ChainorName() string {
	return c.s.f.c.opt.name
}

func (c *TaskContext) ChainorParam() any {
	return c.s.f.c.opt.param
}

func (c *TaskContext) ChainorProps() M {
	return c.s.f.c.opt.props
}

func (c *TaskContext) TaskParam() any {
	return c.s.n.opt.param
}

func (c *TaskContext) TaskProps() M {
	return c.s.n.opt.props
}

func (c *TaskContext) Param() any {
	return c.s.f.opt.param
}

func (c *TaskContext) Props() M {
	return c.s.f.opt.props
}

func (c *TaskContext) future() *future {
	return c.s.f
}
