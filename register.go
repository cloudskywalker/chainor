package chainor

var nodeMap = defalutMapBucket

// Register 全局有效，注册任务函数，通过 NextN 调用
//
// name 任务名称
// task 任务函数，允许设置多个
//
// 注：原本的设计是允许配置优先级的，函数原型为 Register(name string, task TaskFunc, priority ...int)，后决定废除优先级设置，原因有以下几点：
// 1，该包为全异步链路实现，优先级设置的意义不大，完全可以通过拼接链路节点自行实现；
// 2，引入优先级之后，不仅会增加场景的复杂度，且会在多种场景下有歧义，例如 WithParallelFunc、WithAnyPassed 时，高低优先级任务函数具体是如何运转的等等问题；
// 另外，虽然废除了优先级设置，但保留了当前已经实现的部分优先级功能，该部分功能不影响现有机制
func Register(name string, task ...TaskFunc) {
	p := P_Normal

	if len(task) > 0 {
		nodeMap.put(name, &node{
			calls:    task,
			priority: p,
		})
	}
}
