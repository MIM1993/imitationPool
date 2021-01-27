package imitationPool

import (
	"errors"
	"time"
)

var (
	// errQueueIsFull will be returned when the worker queue is full.
	errQueueIsFull = errors.New("the queue is full")

	errQueueISReleased = errors.New("the queue length is zero")
)

//一级缓存pool需要的实现的接口   loop/stack
type workerArray interface {
	len() int
	isEmpty() bool
	insert(worker *goworker) error
	detach() *goworker
	retrieveExpiry(duration time.Duration) []*goworker
	reset()
}

type arrayType int

const (
	stackType arrayType = 1 << iota
	loopQueueType
)

func newWorkerArray(atype arrayType, size int) workerArray {
	switch atype {
	case stackType:
		return newWorkerStack(size)
	case loopQueueType:
		return nil
	default:
		return newWorkerStack(size)
	}
}
