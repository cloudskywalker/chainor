package chainor

import (
	"sync"

	"go.linecorp.com/garr/queue"
	"golang.org/x/sys/cpu"
)

type (
	mapBucket interface {
		put(bkey any, bvalue any)
		exist(bkey any) bool
		eachKey(func(bkey any))
		iteratorB(bkey any) iterator
		lenB(bkey any) int32
	}

	syncMapBucket struct {
		_  cpu.CacheLinePad
		rw sync.RWMutex
		_  cpu.CacheLinePad
		mp map[any]any
		_  cpu.CacheLinePad
	}

	iterator interface {
		HasNext() bool
		Next() any
	}
)

var defalutMapBucket = newSyncMapBucket()

func newSyncMapBucket() mapBucket {
	return &syncMapBucket{
		mp: make(map[any]any),
	}
}

func (m *syncMapBucket) put(bkey any, bvalue any) {
	m.rw.Lock()
	defer m.rw.Unlock()

	if v, ok := m.mp[bkey]; ok {
		v.(queue.Queue).Offer(bvalue)
	} else {
		q := queue.DefaultQueue()
		q.Offer(bvalue)
		m.mp[bkey] = q
	}
}

func (m *syncMapBucket) iteratorB(bkey any) iterator {
	m.rw.RLock()
	defer m.rw.RUnlock()

	if v, ok := m.mp[bkey]; ok {
		// mapBucket 没有 remove 操作，且 queue 为无锁队列，故 return 后的操作不会有并发问题
		return v.(queue.Queue).Iterator()
	}
	return queue.DefaultQueue().Iterator()
}

func (m *syncMapBucket) lenB(bkey any) int32 {
	m.rw.RLock()
	defer m.rw.RUnlock()

	if v, ok := m.mp[bkey]; ok {
		return v.(queue.Queue).Size()
	}
	return 0
}

func (m *syncMapBucket) eachKey(cb func(bkey any)) {
	m.rw.RLock()
	defer m.rw.RUnlock()

	for k := range m.mp {
		cb(k)
	}
}

func (m *syncMapBucket) exist(bkey any) bool {
	m.rw.RLock()
	defer m.rw.RUnlock()

	if _, ok := m.mp[bkey]; ok {
		return true
	}
	return false
}
