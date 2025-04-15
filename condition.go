package chainor

import (
	"go.linecorp.com/garr/queue"
)

type (
	// Predicate 返回 comparable 类型，用以 Case 匹配
	Predicate func(lastResult []any) (result any)

	// Case case 分支的回调函数，回调函数内只能使用参数 c 注册分支链路
	Case func(c *Chainor)

	caseWrap struct {
		expect   any
		caseFunc any
		opt      []TaskOption

		c *Chainor
	}

	switchCase struct {
		predicate Predicate
		cases     []*caseWrap

		c *Chainor
	}

	switchCase2 struct {
		*switchCase
	}
)

// Case 条件节点下的 case 分支，当 Switch 的返回值等于 expect 时执行该分支
//
// expect 为 comparable 类型
// task 为任务函数
// withFunc 为可选项
//
// 注意！！ 所有 Case 和 Default 的 TaskOption 里用户自定义 WithSkipped 均不生效
func (s *switchCase) Case(expect any, task TaskFunc, withFunc ...TaskOption) *switchCase {
	s.cases = append(s.cases, &caseWrap{
		expect:   expect,
		caseFunc: task,
		opt:      withFunc,
	})
	return s
}

// CaseN 条件节点下的 case 分支，当 Switch 的返回值等于 expect 时执行该分支
//
// expect 为 comparable 类型
// name 为 Register 注册的任务名
// withFunc 为可选项
func (s *switchCase) CaseN(expect any, name string, withFunc ...TaskOption) *switchCase {
	s.cases = append(s.cases, &caseWrap{
		expect:   expect,
		caseFunc: name,
		opt:      withFunc,
	})
	return s
}

func (s *switchCase) defaultF() *Chainor {
	var expect any
	caseLen := len(s.cases)
	matched := false

	for i := 1; i <= caseLen; i++ {
		v := s.cases[i-1]

		index := i
		v.opt = append(v.opt, WithSkipped(func(result []any) bool {
			if expect == nil {
				expect = s.predicate(result)
			}
			// 匹配到 case 或 default 的任一条件，其他跳过
			if !matched &&
				(index == caseLen || v.expect == expect) {
				matched = true
				return false
			}
			return true
		}))

		switch obj := v.caseFunc.(type) {
		case TaskFunc:
			s.c.Next(obj, v.opt...)
		case string:
			s.c.NextN(obj, v.opt...)
		}
	}
	return s.c
}

// Default Switch 语法必须以 Default 或 DefaultN 结束，否则 Switch 不会执行
//
// 注意！！所有 Case 和 Default 的 TaskOption 里的用户自定义 WithSkipped 均不生效
func (s *switchCase) Default(task TaskFunc, withFunc ...TaskOption) *Chainor {
	return s.Case(nil, task, withFunc...).defaultF()
}

// DefaultN Switch 语法必须以 Default 或 DefaultN 结束，否则 Switch 不会执行
func (s *switchCase) DefaultN(name string, withFunc ...TaskOption) *Chainor {
	return s.CaseN(nil, name, withFunc...).defaultF()
}

func (s *switchCase2) newChainor() *Chainor {
	nc := s.c.clone()
	nc.queue = queue.DefaultQueue()
	return nc
}

func (s *switchCase2) newFuture(f *future, nc *Chainor) *future {
	nf := f.clone()
	nf.c = nc
	return nf
}

// Case 条件节点下的 case 分支
//
// expect 为 comparable 类型
// cb 为注册的分支链路，当 Switch2 的返回值等于 expect 时执行该链路
//
// 注意！！cb 回调内仅支持注册节点任务，即只允许调用 Next、NextN
func (s *switchCase2) Case(expect any, cb Case) *switchCase2 {
	s.cases = append(s.cases, &caseWrap{
		expect:   expect,
		caseFunc: cb,
	})
	return s
}

// Default Switch2 语法必须以 Default 结束，否则 Switch2 不会执行
//
// cb 同 Case 里的注释说明
//
// 注意！！Default 后原链路彻底分叉，不可再对原 Chainor 继续 Next
func (s *switchCase2) Default(cb Case) {
	s.Case(nil, cb)

	for _, v := range s.cases {
		v.c = s.newChainor()
		v.caseFunc.(Case)(v.c)
	}
	caseLen := len(s.cases)

	s.c.Next(func(ctx *TaskContext, lastResult []any) (any, error) {
		var err error
		expect := s.predicate(lastResult)

		for i := 1; i <= caseLen; i++ {
			v := s.cases[i-1]

			// 匹配到 case 或 default 的任一条件，其他跳过
			if i == caseLen || v.expect == expect {
				s.newFuture(ctx.future(), v.c).forward(lastResult...)
				err = errNotDone
				break
			}
		}

		return nil, err
	}, withSkipReuslt())
}
