package imitationPool

import (
	"time"
)

//stack存储数据结构 一级缓存
type WorkerStack struct {
	//可用的worker队列
	iterms []*goworker
	//过期的队列
	expiry []*goworker
	//初始化可用队列size
	size int
}

func newWorkerStack(size int) *WorkerStack {
	return &WorkerStack{
		iterms: make([]*goworker, 0, size),
		size:   size,
	}
}

func (wq *WorkerStack) len() int {
	return len(wq.iterms)
}

func (wq *WorkerStack) isEmpty() bool {
	return len(wq.iterms) == 0
}

func (wq *WorkerStack) insert(worker *goworker) error {
	wq.iterms = append(wq.iterms, worker)
	return nil
}

func (wq *WorkerStack) detach() *goworker {
	l := wq.len()
	if l == 0 {
		return nil
	}

	w := wq.iterms[l-1]
	wq.iterms = wq.iterms[:l-1]

	return w
}

func (wq *WorkerStack) retrieveExpiry(duration time.Duration) []*goworker {
	n := wq.len()
	if n == 0 {
		return nil
	}
	//过期时间
	expiryTime := time.Now().Add(-duration)
	index := wq.binarySearch(0, n-1, expiryTime)

	wq.expiry = wq.expiry[:0]
	if index != -1 { //-1代表items中worker全部过期
		wq.expiry = append(wq.expiry, wq.iterms[0:index+1]...)
		m := copy(wq.iterms, wq.iterms[index+1:])
		wq.iterms = wq.iterms[:m]
	}

	return wq.expiry
}

func (wq *WorkerStack) binarySearch(l, r int, expiryTime time.Time) int {
	var mid int
	for l < r {
		mid := (l + r) / 2
		if expiryTime.Before(wq.iterms[mid].recycleTime) {
			r = mid - 1
		} else {
			l = mid + 1
		}
	}
	return mid
}

//清空worker队列,并且将worker逐一关闭
func (wq *WorkerStack) reset() {
	for i := 0; i < wq.len(); i++ {
		wq.iterms[i].task <- nil
	}
	wq.iterms = wq.iterms[:0]
}
