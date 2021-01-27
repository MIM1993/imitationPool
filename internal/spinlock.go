package internal

import (
	"runtime"
	"sync"
	"sync/atomic"
)

type spinLock uint32

//加锁
func (sl *spinLock) Lock() {
	for !atomic.CompareAndSwapUint32((*uint32)(sl), 0, 1) {
		runtime.Gosched()
	}
}

//解锁
func (sl *spinLock) Unlock() {
	atomic.StoreUint32((*uint32)(sl), 0)
}

//创建锁
func NewSpinLock() sync.Locker {
	return new(spinLock)
}
