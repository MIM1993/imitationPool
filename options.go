package imitationPool

import (
	"time"
)

//Option 是一个函数 函数式编程
type Option func(opts *Options)

//Options包含在实例化一个ant池时应用的所有选项
type Options struct {
	//worker 过期时间
	ExpiryDuration time.Duration

	//初始化pool时是否开辟内存,进行内存预分配
	PreAlloc bool

	// goroutine阻塞池的最大数目。
	// 0(默认值)表示没有这种限制。
	MaxBlockingTasks int

	//从pool中获取goroutine是否是阻塞的，当为true时，不会阻塞，并且MaxBlockingTasks是无效的
	Nonblocking bool

	//panic处理函数
	// PanicHandler用于处理来自每个worker goroutine的恐慌。
	//如果为nil, panic将再次从worker goroutines被抛出。
	PanicHandle func(interface{})

	//日志记录器是用于记录信息的自定义日志记录器，如果没有设置，
	//默认的标准日志从日志包使用。
	Logger Logger
}

func WithOptions(options *Options) Option {
	return func(opts *Options) {
		opts = options
	}
}

func WithExpiryDuration(expiryDuration time.Duration) Option {
	return func(opts *Options) {
		opts.ExpiryDuration = expiryDuration
	}
}

func WithPreAlloc(preAlloc bool) Option {
	return func(opts *Options) {
		opts.PreAlloc = preAlloc
	}
}

func WithMaxBlockingTasks(maxBlockingTasks int) Option {
	return func(opts *Options) {
		opts.MaxBlockingTasks = maxBlockingTasks
	}
}

func WithNonblocking(nonblocking bool) Option {
	return func(opts *Options) {
		opts.Nonblocking = nonblocking
	}
}

func WithPanicHandle(panicHandle func(interface{})) Option {
	return func(opts *Options) {
		opts.PanicHandle = panicHandle
	}
}

func WithLogger(logger Logger) Option {
	return func(opts *Options) {
		opts.Logger = logger
	}
}

//加载配置
func loadOptions(opt ...Option) *Options {
	opts := new(Options)
	for _, optionFunc := range opt {
		optionFunc(opts)
	}
	return opts
}
