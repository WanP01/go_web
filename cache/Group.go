package cache

import (
	"fmt"
	"log"
	"sync"
)

// Getter 如果缓存不存在，应从数据源（文件，数据库等）获取数据并添加到缓存中
type Getter interface {
	GetData(key string) ([]byte, error)
}

// GetterFunc 函数类型实现某一个接口，称之为接口型函数，方便使用者在调用时既能够传入函数作为参数，也能够传入实现了该接口的结构体作为参数。
type GetterFunc func(key string) ([]byte, error)

func (f GetterFunc) GetData(key string) ([]byte, error) {
	return f(key)
}

var (
	locker sync.RWMutex
	groups = make(map[string]*CacheGroup)
)

// CacheGroup Group 是 Cache 最核心的数据结构，负责与用户的交互，并且控制缓存值存储和获取的流程。
type CacheGroup struct {
	name   string
	cache  *cache
	getter Getter
	loader *SingleFlightGroup //用于确保一个key只负责执行一次
}

func NewGroup(name string, cacheBytes int64, getter Getter) *CacheGroup {
	if getter == nil {
		panic("nil Getter")
	}

	locker.Lock()
	defer locker.Unlock()

	g := &CacheGroup{
		name:   name,
		getter: getter,
		cache:  NewCache(nil, cacheBytes),
		loader: &SingleFlightGroup{},
	}
	groups[name] = g
	return g
}

func GetGroup(name string) *CacheGroup {
	locker.RLocker()
	g, ok := groups[name]
	defer locker.RUnlock()
	if !ok {
		return nil
	}
	return g
}

// Get 从cache 中拿取对应缓存
// 找到直接返回，没找到从数据库搜索数据，返回并存储再缓存中
func (g *CacheGroup) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is requried")
	}

	if v, ok := g.cache.Get(key); ok {
		log.Println("[Cache] hit")
		return v, nil
	}
	// 如果花奴才能内部没有，调用Getter从数据源拿取消息
	return g.load(key)
}

func (g *CacheGroup) load(key string) (value ByteView, err error) {

	onceGetData, err := g.loader.Do(key, func() (interface{}, error) {
		return g.getLocally(key)
	})
	if err == nil {
		log.Println("Getter load success")
		return onceGetData.(ByteView), nil
	}
	return
}

func (g *CacheGroup) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.GetData(key)
	if err != nil {
		return ByteView{}, err

	}
	value := ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, value)
	return value, nil
}

func (g *CacheGroup) populateCache(key string, value ByteView) {
	g.cache.Set(key, value)
}
