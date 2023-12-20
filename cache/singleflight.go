package cache

import "sync"

type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

type SingleFlightGroup struct {
	mu sync.Mutex // 用于map并发读写操作加锁
	m  map[string]*call
}

func (g *SingleFlightGroup) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}
	c := &call{}
	c.wg.Add(1) // 需要在释放mu 锁之前加wg
	g.m[key] = c
	g.mu.Unlock()

	// 向数据库获取数据的操作
	c.val, c.err = fn()
	c.wg.Done()

	// 获取完后将注册的call销毁
	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()

	return c.val, c.err
}
