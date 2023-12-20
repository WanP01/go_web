package cache

import "sync"

type Cache interface {
	Get(key string) (value Value, ok bool)
	Set(key string, value Value)
	Del(key string)
	RemoveOldest()
	Len() int
}

type cache struct {
	RWM           sync.RWMutex
	cacheStrategy Cache
	cacheBytes    int64
}

func NewCache(strategy Cache, cacheBytes int64) *cache {
	c := &cache{
		RWM:           sync.RWMutex{},
		cacheStrategy: strategy,
		cacheBytes:    cacheBytes,
	}
	return c
}

// Set 对外暴露数据以ByteView格式
func (c *cache) Set(key string, value ByteView) {
	c.RWM.Lock()
	defer c.RWM.Unlock()
	// 默认实现LRU算法
	//  方法中，判断了 c.cacheStrategy 是否为 nil，如果等于 nil 再创建实例。
	// 这种方法称之为延迟初始化(Lazy Initialization)，一个对象的延迟初始化意味着该对象的创建将会延迟至第一次使用该对象时。主要用于提高性能，并减少程序内存要求。
	if c.cacheStrategy == nil {
		c.cacheStrategy = NewLRU(c.cacheBytes, nil)
	}
	c.cacheStrategy.Set(key, value)
}

// Get 对外暴露数据以ByteView格式
func (c *cache) Get(key string) (value ByteView, ok bool) {
	c.RWM.RLock()
	defer c.RWM.RUnlock()
	// 默认实现LRU算法
	if c.cacheStrategy == nil {
		c.cacheStrategy = NewLRU(c.cacheBytes, nil)
	}
	if v, ok := c.cacheStrategy.Get(key); ok {
		return v.(ByteView), true
	}
	return ByteView{}, false
}
