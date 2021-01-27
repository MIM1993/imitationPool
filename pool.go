package imitationPool

import (
	"github.com/MIM1993/imitationPool/internal"
	"sync"
	"sync/atomic"
	"time"
)

//pool 从客户端接收任务，回收goroutines来控制goroutines数量
type Pool struct {
	//pool容量
	capacity int32

	//正在运行的goroutines数量
	running int32

	//存储用于存放正在运行的worker的slice 一级缓存
	workers workerArray //接口类型

	//pool的状态 open or close
	state int32

	//lock 用于同步操作, 自实现spining lock
	lock sync.Locker

	//用于唤醒处于沉睡状态的goroutine 惊群
	cond *sync.Cond

	//缓冲池 用来做快速创建goroutine的工厂和二级缓存
	workerCache sync.Pool

	//pool中已经被阻塞的groutines数量
	blockingNum int

	//基本配置
	options *Options
}

//定期清除过期worker
func (p *Pool) periodicallyPurge() {
	heartbate := time.NewTicker(p.options.ExpiryDuration)
	defer heartbate.Stop()

	for range heartbate.C {
		//检测pool是否关闭,当关闭时退出
		if atomic.LoadInt32(&p.state) == CLOSED {
			break
		}

		p.lock.Lock()
		expiredWorkers := p.workers.retrieveExpiry(p.options.ExpiryDuration)
		p.lock.Unlock()

		//向task channel中发送nil,使worker触发关闭机制
		for i := range expiredWorkers {
			expiredWorkers[i].task <- nil
		}

		//唤醒沉睡中的goroutine,惊群效应
		if p.running == 0 {
			p.cond.Broadcast()
		}

	}
}

//创建一个pool对象
func NewPool(size int, options ...Option) (*Pool, error) {
	//加载配置数据
	opts := loadOptions(options...)

	if size <= 0 {
		size = -1
	}

	if expiry := opts.ExpiryDuration; expiry < 0 {
		return nil, ErrInvalidPoolExpiry
	} else if expiry == 0 {
		opts.ExpiryDuration = DefaultCleanIntervalTime
	}

	if opts.Logger == nil {
		opts.Logger = defaultLogger
	}

	p := &Pool{
		capacity: int32(size),
		options:  opts,
		lock:     internal.NewSpinLock(),
	}

	//创建二级缓存池
	p.workerCache.New = func() interface{} {
		return &goworker{
			pool: p,
			task: make(chan func(), workerCap),
		}
	}

	//初始化空间
	if p.options.PreAlloc {
		if size == -1 {
			return nil, ErrInvalidPreAllocSize
		}
		p.workers = newWorkerArray(loopQueueType, size)
	} else {
		p.workers = newWorkerArray(stackType, size)
	}

	//初始化信号量
	p.cond = sync.NewCond(p.lock)

	//开启清除过期goroutine任务
	p.periodicallyPurge()

	return p, nil
}

// Submit submits a task to this pool.
func (p *Pool) Submit(task func()) error {
	if atomic.LoadInt32(&p.state) == CLOSED {
		return ErrPoolClosed
	}

	//todo: 获取一个worker
	var w *goworker
	w = p.retrieveWorker()
	if w == nil {
		return ErrPoolOverload
	}

	//如果worker正常,则将任务放进channel
	w.task <- task

	return nil
}

//返回运行worker数量
func (p *Pool) Running() int {
	return int(atomic.LoadInt32(&p.running))
}

//返回平pool的容量
func (p *Pool) Cap() int {
	return int(atomic.LoadInt32(&p.capacity))
}

//返回可用的worker数量
func (p *Pool) Free() int {
	return p.Cap() - p.Running()
}

//Tune changes the capacity of this pool, this method is noneffective to the infinite pool.
func (p *Pool) Tune(size int) {
	if capNum := p.Cap(); capNum == -1 ||
		size <= 0 ||
		size == capNum ||
		p.options.PreAlloc {
		return
	}
	//更新容量
	atomic.StoreInt32(&p.capacity, int32(size))
}

//释放pool(关闭)
func (p *Pool) Release() {
	atomic.StoreInt32(&p.state, CLOSED)
	p.lock.Lock()
	p.workers.reset()
	p.lock.Unlock()
}

//重启
func (p *Pool) Reboot() {
	if atomic.CompareAndSwapInt32(&p.state, CLOSED, OPENED) {
		p.periodicallyPurge()
	}
}

func (p *Pool) incRunning() {
	atomic.AddInt32(&p.state, 1)
}

func (p *Pool) decRunning() {
	atomic.AddInt32(&p.state, -1)
}

func (p *Pool) retrieveWorker() (w *goworker) {
	spawnWorker := func() {
		//从二级缓存池中获取一个worker
		w = p.workerCache.Get().(*goworker)
		//启动这个worker
		w.run()
	}

	p.lock.Lock()

	//先从一级缓存获取
	w = p.workers.detach()
	if w != nil {
		p.lock.Unlock()
	} else if capacity := p.Running(); capacity == -1 {
		p.lock.Unlock()
		spawnWorker()
	} else if p.Running() < capacity {
		p.lock.Unlock()
		spawnWorker()
	} else {
		if p.options.Nonblocking {
			p.lock.Unlock()
			return
		}
	Reentry:
		if p.options.MaxBlockingTasks != 0 && p.blockingNum < p.options.MaxBlockingTasks {
			p.lock.Unlock()
			return
		}
		p.blockingNum++
		p.cond.Wait()
		p.blockingNum--
		if p.Running() == 0 {
			p.lock.Unlock()
			spawnWorker()
			return
		}

		w = p.workers.detach()
		if w == nil {
			goto Reentry
		}

		p.lock.Unlock()
	}

	return
}

//将worker返还pool
func (p *Pool) revertWorker(w *goworker) bool {
	if capacity := p.Cap(); (capacity > 0 && p.Running() > capacity) || atomic.LoadInt32(&p.state) == CLOSED {
		return false
	}

	//返回时间,用于进行过期检测
	w.recycleTime = time.Now()
	p.lock.Lock()

	if err := p.workers.insert(w); err != nil {
		p.lock.Unlock()
		return false
	}

	//通过信号量唤醒一个goroutine
	p.cond.Signal()
	p.lock.Unlock()
	return true
}
