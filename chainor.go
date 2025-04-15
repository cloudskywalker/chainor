// MIT License

// Copyright (c) 2022 QinZhen

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package chainor

import (
	"sort"

	"go.linecorp.com/garr/queue"
)

// NewChainor 返回一个 chainor 实例
func NewChainor(withFunc ...ChainorOption) *Chainor {
	return &Chainor{
		opt:   mergeOption[ChainorOption](withFunc...),
		queue: queue.DefaultQueue(),
	}
}

// Next 注册链路上的节点任务函数
func (c *Chainor) Next(task TaskFunc, withFunc ...TaskOption) *Chainor {
	if task == nil {
		return c
	}

	c.queue.Offer((&node{
		opt:   mergeOption[TaskOption](withFunc...),
		calls: TaskFuncs{task},
	}).concrete())

	return c
}

// collateNode 该函数主要是对优先级进行排序和归类，目前优先级机制已废除，不过还是保留了这部分代码，阅读代码时可略过该部分
func collateNode(name string, cb func(withfuncs TaskFuncs)) {
	tempMap := newSyncMapBucket()
	keyArray := make([]int, 0, nodeMap.lenB(name))

	iter := nodeMap.iteratorB(name)
	for iter.HasNext() {
		n := iter.Next().(*node)
		tempMap.put(n.priority, n)
	}

	tempMap.eachKey(func(bkey any) {
		keyArray = append(keyArray, bkey.(int))
	})
	sort.SliceStable(keyArray, func(i, j int) bool {
		return keyArray[i] > keyArray[j]
	})

	for i := 0; i < len(keyArray); i++ {
		iter = tempMap.iteratorB(keyArray[i])
		var withs TaskFuncs

		for iter.HasNext() {
			n := iter.Next().(*node)
			withs = append(withs, n.calls...)
		}
		cb(withs)
	}
}

// NextN 通过任务名称注册链路上的节点任务，name 为 Register 注册的任务名
func (c *Chainor) NextN(name string, withFunc ...TaskOption) *Chainor {
	if !nodeMap.exist(name) {
		return c
	}

	// 优先级机制已废除，代码保留，此处所有任务的优先级都是相同的
	collateNode(name, func(tasks TaskFuncs) {
		withs := withFunc
		if len(tasks) > 1 {
			withs = append(withFunc, WithParallelFunc(tasks[1:]...))
		}
		c.Next(tasks[0], withs...)
	})
	return c
}

func (c *Chainor) clone() *Chainor {
	nc := *c
	return &nc
}

// Switch 条件节点，与 Switch2 不同的是此为单节点分支而非链路分支，可分可合
func (c *Chainor) Switch(predicate Predicate) *switchCase {
	return &switchCase{
		predicate: predicate,
		c:         c,
	}
}

// Switch2 条件节点，可根据条件结果执行不同的链路，类似有向无环图（DAG），但只可分不可合
func (c *Chainor) Switch2(predicate Predicate) *switchCase2 {
	return &switchCase2{
		switchCase: c.Switch(predicate),
	}
}

// Invoke 触发 Chainor
//
// onSucess 成功回调，onFailed 失败回调（内置错误定义在 types.go 里，包括超时、未命中等）
func Invoke(c *Chainor, onSuccess func(result []any), onFailed func(err error), withFunc ...Option) {
	c.newFuture(mergeOption[Option](withFunc...), onSuccess, onFailed).forward()
}
