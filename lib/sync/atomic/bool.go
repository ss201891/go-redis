package atomic

import "sync/atomic"

// Go语言标准库中的sync/atomic包提供了偏底层的原子内存原语(atomic memory primitives)，用于实现同步算法，其本质是将底层CPU提供的原子操作指令封装成了Go函数。
// 使用sync/atomic提供的原子操作可以确保在任意时刻只有一个goroutine对变量进行操作，避免并发冲突。
// 原生没有bool 用uint32包装一个bool型的atomic
// Boolean is a boolean value, all actions of it is atomic
type Boolean uint32

// Get reads the value atomically
func (b *Boolean) Get() bool {
	return atomic.LoadUint32((*uint32)(b)) != 0
}

// Set writes the value atomically
func (b *Boolean) Set(v bool) {
	if v {
		atomic.StoreUint32((*uint32)(b), 1)
	} else {
		atomic.StoreUint32((*uint32)(b), 0)
	}
}
