package imitationPool

import "time"

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
	//todo: 更新pool中运行goroutine数量
}
