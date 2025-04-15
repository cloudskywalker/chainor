package chainor

import (
	"sync"

	"github.com/zeromicro/go-zero/core/threading"
)

type (
	future struct {
		opt *option
		c   *Chainor
		ctx *chainorContext

		onSuccess func([]any)
		onFailed  func(error)
	}

	resChan struct {
		c        chan *result
		stopChan chan struct{}

		closeOnce, stopOnce *sync.Once
		rw                  sync.RWMutex // 读多写少（只有一次 close 写），对性能影响很小
		stopFlag            bool
	}

	returnRes struct {
		res            []any
		err            error
		maxCnt, curCnt int
		doneFlag       bool
	}

	step struct {
		n   *node
		f   *future
		ctx *TaskContext

		resChan *resChan

		lastRes []any
		nowRes  *returnRes
	}

	result struct {
		err error
		msg any
	}
)

func (c *Chainor) newFuture(opt *option, onSuccess func([]any), onFailed func(error)) *future {
	f := &future{
		opt:       opt,
		c:         c,
		onSuccess: onSuccess,
		onFailed:  onFailed,
	}
	f.ctx = f.newChainorContext()

	return f
}

func (f *future) clone() *future {
	nf := *f
	return &nf
}

func (f *future) forward(lastRes ...any) {
	threading.GoSafe(func() {
		done := true

		iter := f.c.queue.Iterator()
		for iter.HasNext() {
			n := iter.Next().(*node)

			if res, err := (&step{
				n:       n,
				f:       f,
				lastRes: lastRes,
			}).start(); err == nil && !n.opt.skipResult {
				lastRes = res
			} else {
				if err == errNotDone {
					done = false
				} else if f.onFailed != nil {
					f.onFailed(err)
				}
				goto failed
			}
		}
		if f.onSuccess != nil {
			f.onSuccess(lastRes)
		}

	failed:
		if done && f.ctx.cancel != nil {
			f.ctx.cancel()
		}
	})
}

func newResChan() *resChan {
	c := &resChan{
		c:         make(chan *result),
		stopChan:  make(chan struct{}),
		closeOnce: new(sync.Once),
		stopOnce:  new(sync.Once),
	}
	return c
}

func (r *resChan) response(res *result) {
	r.rw.RLock()
	defer r.rw.RUnlock()

	if !r.stopFlag {
		r.c <- res
	}
}

func (r *resChan) ack(msg any) {
	r.response(&result{
		msg: msg,
	})
}

func (r *resChan) nack(err error) {
	r.response(&result{
		err: err,
	})
}

func (r *resChan) close() {
	r.closeOnce.Do(func() {
		close(r.c)
	})
}

func (r *resChan) stop() {
	r.stopOnce.Do(func() {
		// 通知关闭 channel
		r.stopChan <- struct{}{}
		close(r.stopChan)
	})
}

// wait 监听 close 信号
func (r *resChan) wait() {
	threading.GoSafe(func() {
		for range r.stopChan {
			r.rw.Lock()

			if !r.stopFlag {
				// 遵循 the channel closing principle
				r.close()
				r.stopFlag = true
			}
			r.rw.Unlock()
		}
	})
}

func (r *returnRes) store(res any) {
	r.res = append(r.res, res)
}

func (r *returnRes) done() {
	r.doneFlag = true
}

func (r *returnRes) isDone() bool {
	return r.doneFlag
}

func (r *returnRes) withError(err error) {
	r.err = err
}

func (r *returnRes) error() error {
	return r.err
}

func (r *returnRes) result() []any {
	return r.res
}

func (r *returnRes) isFull() bool {
	return r.curCnt >= r.maxCnt
}

func (r *returnRes) inc() {
	r.curCnt++
}

func (s *step) newReutrnRes() {
	s.nowRes = &returnRes{}

	switch s.n.opt.cct.mode() {
	case workerpoolM:
		s.nowRes.maxCnt = len(s.n.calls)
	case parallelM:
		s.nowRes.maxCnt = len(s.n.calls) * s.n.opt.cct.count()
	default:
		s.nowRes.maxCnt = len(s.n.calls)
	}
	s.nowRes.res = make([]any, 0, s.nowRes.maxCnt)
}

func (s *step) rangeCalls(callback func(taskFunc TaskFunc)) {
	for i := 0; i < len(s.n.calls); i++ {
		callback(s.n.calls[i])
	}
}

func (s *step) call(call TaskFunc) {
	if res, err := call(s.newTaskContext(), s.lastRes); err != nil {
		s.resChan.nack(err)
	} else {
		s.resChan.ack(res)
	}
}

func (s *step) runWithWorkerPool() {
	s.rangeCalls(func(call TaskFunc) {
		if err := s.n.workerpool.Submit(func() {
			s.call(call)
		}); err != nil {
			s.resChan.nack(err)
		}
	})
}

func (s *step) runWithParallel() {
	s.rangeCalls(func(call TaskFunc) {
		for j := 0; j < s.n.opt.cct.count(); j++ {
			threading.GoSafe(func() {
				s.call(call)
			})
		}
	})
}

func (s *step) runCalls() {
	s.rangeCalls(func(call TaskFunc) {
		threading.GoSafe(func() {
			s.call(call)
		})
	})
}

func (s *step) init() {
	s.resChan = newResChan()
	s.newReutrnRes()

	s.resChan.wait()
}

func (s *step) decide(res *result) bool {
	if s.nowRes.isDone() || s.nowRes.isFull() {
		return true
	}

	done := func() bool {
		s.nowRes.done()
		return true
	}
	s.nowRes.inc()

	// 期望的结果是否已收到
	switch {
	case s.n.opt.anyPassedFunc != nil:
		if res.err == nil && s.n.opt.anyPassedFunc(res.msg) {
			s.nowRes.store(res.msg)
			return done()
		}
		if s.nowRes.isFull() {
			s.nowRes.withError(ErrNoPassed)
			return done()
		}
	default:
		if res.err != nil {
			s.nowRes.withError(res.err)
			return done()
		}
		s.nowRes.store(res.msg)
		if s.nowRes.isFull() {
			return done()
		}
	}
	return false
}

func (s *step) wait() ([]any, error) {
FOR:
	for {
		select {
		case <-s.f.ctx.ctx.Done():
			s.nowRes.withError(ErrTimeout)
			s.resChan.stop()
			break FOR
		case value, ok := <-s.resChan.c:
			if !ok {
				break FOR
			}
			if s.decide(value) {
				s.resChan.stop()
			}
		}
	}

	// 等待 close，防止泄露，此处不会持久阻塞，close 会立马到来
	if s.nowRes.error() == ErrTimeout {
		for range s.resChan.c {
		}
	}
	return s.nowRes.result(), s.nowRes.error()
}

func (s *step) run() ([]any, error) {
	s.init()

	switch s.n.opt.cct.mode() {
	case workerpoolM:
		s.runWithWorkerPool()
	case parallelM:
		s.runWithParallel()
	default:
		s.runCalls()
	}
	return s.wait()
}

func (s *step) start() ([]any, error) {
	if s.n.opt.skippedFunc != nil {
		if s.n.opt.skippedFunc(s.lastRes) {
			return s.lastRes, nil
		}
	}

	return s.run()
}
