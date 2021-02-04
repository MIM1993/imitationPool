package imitationPool

import (
	"runtime"
	"time"
)

type goworker struct {
	//worker所在pool的指针
	pool *Pool

	//接收task的channel
	task chan func()

	//时间标记，当worker被使用后并重新放入队列中时，更新这个字段
	recycleTime time.Time
}

//启动worker，在初始话一个worker时调用
func (g *goworker) run() {
	//更新pool中运行goroutine数量 +1
	g.pool.incRunning()
	go func() {
		defer func() {
			//更新pool中运行goroutine数量 -1
			g.pool.decRunning()
			//将关闭的worker重新放入二级缓存 pool
			g.pool.workerCache.Put(g)
			if perr := recover(); perr != nil {
				if ph := g.pool.options.PanicHandle; ph != nil {
					ph(perr)
				} else {
					g.pool.options.Logger.Printf("worker exits from a panic: %v\n", perr)
					//var buf [4096]byte
					var buf []byte = make([]byte, 4096, 4096)
					n := runtime.Stack(buf, false)
					g.pool.options.Logger.Printf("worker exits from a panic: %v\n", string(buf[:n]))
				}
			}
		}()

		//循环监听task channel
		for t := range g.task {
			if t == nil {
				return
			}
			//调用任务功能函数
			t()
			//回收worker近一级缓存
			if ok := g.pool.revertWorker(g); !ok {
				return
			}
		}

	}()
}
