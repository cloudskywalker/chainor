# chainor

## 简介
chainor，源于 Chain of Responsibility 的缩写，一个轻量级嵌入式任务调度框架，融合了责任链和 promise 机制，并进行了全新设计

## 原理

- 节点任务以 goroutine 的方式异步运行；
- 节点任务之间通过 channel 进行通讯； 
- 允许链路分支，即可以像 DAG（有向无环图） 一样发散；

## 特性

- 支持链式调用；
- 可以共享任务节点；
- 任务节点允许跳过、并行、分叉等；
- 支持并行度和协程池两种并行模式；
- 支持条件分支，Switch-Case 模式。Switch 与 Switch2 的区别请详细阅读代码注释及单元测试示例；

## 用法示例

```golang
    import (
        "fmt"
        "sync"
        
        C "github.com/cloudskywalker/chainor"
    )

    wg := sync.WaitGroup{}
    wg.Add(1)

    C.Register("task1", func(ctx *C.TaskContext, lastResult []any) (result any, err error) {
        return 1, nil
    }, func(ctx *C.TaskContext, lastResult []any) (result any, err error) {
        return 2, nil
    })

    chn := C.NewChainor(C.WithChainorName("test"))
    chn.Next(func(ctx *C.TaskContext, lastResult []any) (result any, err error) {
        ctx.WithValue("id", "123")
        return 1, nil
    }).Switch2(func(lastResult []any) (result any) {
        if lastResult[0] == 1 {
            return 2
        }
        return 3
    }).Case(2, func(c1 *C.Chainor) {
        c1.NextN("task1", C.WithTaskParam(5),
                C.WithSkipped(func(result []any) bool {
                    if result[0] == 5 {
                        return true
                    }
                    return false
                }), C.WithWorkerPool(5)).
            Next(func(ctx *C.TaskContext, lastResult []any) (result any, err error) {
				return 10, nil
			}, C.WithParallel(2)).
			Switch(func(lastResult []any) (result any) {
                return lastResult[0]
            }).
            Case(20, func(ctx *C.TaskContext, lastResult []any) (result any, err error) {
                return 30, nil
            }).
            Default(func(ctx *C.TaskContext, lastResult []any) (result any, err error) {
                return 50, nil
            }).
            Next(func(ctx *C.TaskContext, lastResult []any) (result any, err error) {
                return lastResult, nil
            }, C.WithParallelFunc(func(ctx *C.TaskContext, lastResult []any) (result any, err error) {
                return 88, nil
            }), C.WithAnyPassed(func(result any) bool {
                if result != 88 {
                    return true
                }
                return false
            }))
    }).Default(func(c2 *C.Chainor) {
		c2.Next(func(ctx *C.TaskContext, lastResult []any) (result any, err error) {
			return 66, nil
		})
	})

    C.Invoke(chn, func(result []any) {
		fmt.Printf("result should be [[50]]: %v\n", result)
		wg.Done()
    }, nil, C.WithTimeout(2*time.Second), C.WithProps(C.M{"tkey": "tvalue"}))

    wg.Wait()
```

